package controller

import (
	"fmt"
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	apis "metricsadvisor.ai/appservice/apis/multitenancy/v1"
	components "metricsadvisor.ai/appservice/components"
	reconciler "metricsadvisor.ai/appservice/reconciler"
)

var APP_SERVICE_WORK_QUEUE_NAME = "ApiServiceWorkQueue"
var APP_SERVICE_DEPLOYMENT_SUFFIX = "-deployment"
var APP_SERVICE_SERVICE_SUFFIX = "-svc"
var APP_SERVICE_KIND = "AppService"

var RESOURCE_KIND_ROLE = "Role"
var RESOURCE_KIND_CLUSTER_ROLE = "ClusterRole"
var RESOURCE_KIND_ROLE_BINDING = "RoleBinding"
var RESOURCE_KIND_CLUSTER_ROLE_BINDING = "ClusterRoleBinding"

type EventItem struct {
	ClusterId string
	Key       string
}

func (item *EventItem) String() string {
	return fmt.Sprintf("Cluster[%s]-[%s]", item.ClusterId, item.Key)
}

type ApiServiceController struct {
	clusterToolMap map[string]*components.ClusterTool
	workqueue      workqueue.RateLimitingInterface
	replicaTool    reconciler.ReplicaTool
	//TODO learn more about Recorder and leverage it in current program
}

func NewApiServiceController(clusterToolMap map[string]*components.ClusterTool) *ApiServiceController {
	controller := &ApiServiceController{
		clusterToolMap: clusterToolMap,
		workqueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), APP_SERVICE_WORK_QUEUE_NAME),
		replicaTool:    reconciler.NewReplicaTool(&clusterToolMap),
	}

	// Add event handler
	controller.listenToInformer()

	return controller
}

func (c *ApiServiceController) listenToInformer() {
	// These two function is used to generate callback for each cluster's apiservice informer.
	// See more about go's closure
	generateAddEnqueueFunc := func(clusterId string) func(obj interface{}) {
		return func(obj interface{}) {
			c.enqueueKDeployment(clusterId, obj)
		}
	}

	generateUpdateEnqueueFunc := func(clusterId string) func(old, new interface{}) {
		return func(old, new interface{}) {
			c.enqueueKDeployment(clusterId, new)
		}
	}

	for clusterId, clusterTool := range c.clusterToolMap {
		clusterTool.ApiServiceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    generateAddEnqueueFunc(clusterId),
			UpdateFunc: generateUpdateEnqueueFunc(clusterId),
		})
	}
}

func (c *ApiServiceController) enqueueKDeployment(clusterId string, obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(EventItem{
		ClusterId: clusterId,
		Key:       key,
	})
}

func (c *ApiServiceController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Info("Start App Service Controller")
	for i := 0; i < threadiness; i++ {
		klog.Infof("Starting worker %d", i)
		// TODO: Use wait.Until is a bit confusing here because runworker won't exits
		// Find a better way to do that
		go wait.Until(c.runWorker, time.Second, stopCh)
		klog.Infof("Worker %d started", i)
	}

	// we need to listen from this channel to avoid that the main thread exits
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

func (c *ApiServiceController) runWorker() {
	for c.fetchEventItemAndProcess() {
	}
}

func (c *ApiServiceController) fetchEventItemAndProcess() bool {

	eventItem, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var item EventItem
		var ok bool
		if item, ok = obj.(EventItem); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected EventItem in workqueue but got %#v", obj))
			return nil
		}
		err := c.processOneEventItem(item)
		if err != nil {
			c.workqueue.AddRateLimited(item)
			return fmt.Errorf("error when processing EventItem %s, %s", item.ClusterId, item.Key)
		}
		// If the process of the event item is failed, we won't invoke Forget() for it
		c.workqueue.Forget(obj)
		klog.Infof("Successfully process event item %s, %s", item.ClusterId, item.Key)
		return nil
	}(eventItem)

	// If error occurs, the process will still continue
	if err != nil {
		klog.Errorf("Error when processing app service event item %s", eventItem)
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *ApiServiceController) processOneEventItem(item EventItem) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(item.Key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", item.Key))
	}

	// Pick the apiService resource in Lister from corresponding Cluster
	appService, err := c.clusterToolMap[item.ClusterId].GetApiService(namespace, name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("ApiService '%s' in work queue no longer exits", item))
		}
		return err
	}

	// First, Sync the target app service across all cluster
	if err := c.syncApiServiceToAllCluster(appService); err != nil {
		utilruntime.HandleError(err)
		klog.Error(err, "Error when syncing app service across cluster.")
		return err
	}

	// Then, main sub resources in each cluster

	// Maintain deployment
	deploymentReconciler := reconciler.NewDeploymentReconciler(appService, c.clusterToolMap)
	if err := reconciler.Reconcile(deploymentReconciler); err != nil {
		utilruntime.HandleError(err)
		return err
	}

	// Maintain Service
	serviceReconciler := reconciler.NewServiceReconciler(appService, c.clusterToolMap)
	if err := reconciler.Reconcile(serviceReconciler); err != nil {
		utilruntime.HandleError(err)
		return err
	}

	// Maintain Role
	// TOOD: think about how to delete expired resouce if the kind of Role/ClusterRole is changed
	if c.needToMaintainRole(appService) {
		roleReconciler := reconciler.NewRoleReconciler(appService, c.clusterToolMap)
		if err := reconciler.Reconcile(roleReconciler); err != nil {
			utilruntime.HandleError(err)
			return err
		}
	}

	// Maintain ClusterRole
	if c.needToMaintainClusterRole(appService) {
		clusterRoleReconciler := reconciler.NewClusterRoleReconciler(appService, c.clusterToolMap)
		if err := reconciler.Reconcile(clusterRoleReconciler); err != nil {
			utilruntime.HandleError(err)
			return err
		}
	}

	// Maintain RoleBinding
	if c.needToMaintainRoleBinding(appService) {
		roleBindingReconciler := reconciler.NewRoleBindingReconciler(appService, c.clusterToolMap)
		if err := reconciler.Reconcile(roleBindingReconciler); err != nil {
			utilruntime.HandleError(err)
			return err
		}
	}

	// Maintain ClusterRoleBinding
	if c.needToMaintainClusterRoleBinding(appService) {
		clusterRoleBindingReconciler := reconciler.NewClusterRoleBindingReconciler(appService, c.clusterToolMap)
		if err := reconciler.Reconcile(clusterRoleBindingReconciler); err != nil {
			utilruntime.HandleError(err)
			return err
		}
	}

	// Maintain VirtualService
	if c.needToMaintainVirtualService(appService) {
		virtualServiceReconciler := reconciler.NewVirtualServiceReconciler(appService, c.clusterToolMap)
		if err := reconciler.Reconcile(virtualServiceReconciler); err != nil {
			utilruntime.HandleError(err)
			return err
		}
	}

	return nil
}

func (c *ApiServiceController) needToMaintainRole(target *apis.AppService) bool {
	if target.Spec.RoleTemplate.Kind == RESOURCE_KIND_ROLE {
		return true
	}
	return false
}

func (c *ApiServiceController) needToMaintainClusterRole(target *apis.AppService) bool {
	if target.Spec.RoleTemplate.Kind == RESOURCE_KIND_CLUSTER_ROLE {
		return true
	}
	return false
}

func (c *ApiServiceController) needToMaintainRoleBinding(target *apis.AppService) bool {
	if target.Spec.RoleBindingTemplate.Kind == RESOURCE_KIND_ROLE_BINDING {
		return true
	}
	return false
}

func (c *ApiServiceController) needToMaintainClusterRoleBinding(target *apis.AppService) bool {
	if target.Spec.RoleBindingTemplate.Kind == RESOURCE_KIND_CLUSTER_ROLE_BINDING {
		return true
	}
	return false
}

func (c *ApiServiceController) needToMaintainVirtualService(target *apis.AppService) bool {
	if len(target.Spec.VirtualServiceSpec.Hosts) > 0 {
		return true
	}
	return false
}

func (c *ApiServiceController) verifyRoleSpec(target *apis.AppService) error {
	kind := target.Spec.RoleTemplate.Kind
	if kind != RESOURCE_KIND_ROLE || kind != RESOURCE_KIND_CLUSTER_ROLE {
		return fmt.Errorf("Spec for RoleKind is not correct. Value: %s", kind)
	}
	return nil
}

func (c *ApiServiceController) syncApiServiceToAllCluster(target *apis.AppService) error {
	for clusterId, tool := range c.clusterToolMap {
		current, err := tool.GetApiService(target.Namespace, target.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				err := tool.CreateApiService(target)
				if err != nil {
					return err
				}
				klog.Info("create apiService[%s] on cluster [%s]", target.Name, clusterId)
			} else {
				return err
			}
		} else if tool.IsApiServiceDifferent(current, target) {
			if c.shouldUpdate(current, target) {
				klog.Infof("ApiService is different on cluster[%s]. Will Update", clusterId)
				err := tool.UpdateApiService(current, target)
				if err != nil {
					klog.Error(err, fmt.Sprintf("Failed to update app service[%s] on cluster[%s]", target.Name, clusterId))
				} else {
					klog.Infof("update app service[%s] on cluster [%s]", current.Name, clusterId)
				}
			}
		}
	}

	return nil
}

func (c *ApiServiceController) shouldUpdate(current, target *apis.AppService) bool {
	if current.ObjectMeta.ManagedFields == nil {
		return true
	}
	updateTimeInCluster := components.GetResourceUpdateTime(current.ObjectMeta)
	updateTimeOfTarget := components.GetResourceUpdateTime(target.ObjectMeta)
	if updateTimeInCluster == nil && updateTimeOfTarget != nil {
		return true
	}
	if updateTimeInCluster.Before(updateTimeOfTarget) {
		return true
	}
	return false
}

func (c *ApiServiceController) getDeploymentName(appService *apis.AppService) string {
	return appService.Name + APP_SERVICE_DEPLOYMENT_SUFFIX
}

func (c *ApiServiceController) getServiceName(appService *apis.AppService) string {
	return appService.Name + APP_SERVICE_SERVICE_SUFFIX
}

func (c *ApiServiceController) newDeployment(appService *apis.AppService, replicas *int32) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.getDeploymentName(appService),
			Namespace: appService.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(appService, apis.SchemeGroupVersion.WithKind(APP_SERVICE_KIND)),
			},
		},
		Spec: appService.Spec.DeploymentSpec,
	}
	deployment.Spec.Replicas = replicas
	return deployment
}

// TODO: verify whether this comparision is enough or overhead
func (c *ApiServiceController) isDeploymentDifferent(target, current *appsv1.Deployment) bool {
	if *target.Spec.Replicas != *current.Spec.Replicas {
		return true
	}
	if !reflect.DeepEqual(target.Spec.Selector, current.Spec.Selector) {
		return true
	}
	if !reflect.DeepEqual(target.Spec.Template, current.Spec.Template) {
		return true
	}
	return false
}
