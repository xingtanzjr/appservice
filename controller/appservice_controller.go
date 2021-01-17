package controller

import (
	"fmt"
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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
	apiService, err := c.clusterToolMap[item.ClusterId].GetApiService(namespace, name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("ApiService '%s' in work queue no longer exits", item))
		}
		return err
	}

	// First, Sync the target app service across all cluster
	if err := c.syncApiServiceToAllCluster(apiService); err != nil {
		utilruntime.HandleError(err)
		klog.Error(err, "Error when syncing app service across cluster.")
		return err
	}

	// Then, main sub resources in each cluster

	// // Maintain deployment
	// if err := c.maintainDeployment(apiService); err != nil {
	// 	utilruntime.HandleError(err)
	// 	klog.Error(err, fmt.Sprintf("Error when maintain deployments for AppService[%s]", apiService.Name))
	// 	return err
	// }

	deploymentReconciler := reconciler.NewDeploymentReconciler(*apiService, c.clusterToolMap)
	if err := reconciler.Reconcile(deploymentReconciler); err != nil {
		utilruntime.HandleError(err)
		return err
	}

	// Maintain Service
	if err := c.maintainService(apiService); err != nil {
		utilruntime.HandleError(err)
		klog.Error(err, fmt.Sprintf("Error when maintain service for AppService[%s]", apiService.Name))
		return err
	}

	if err := c.maintainRole(apiService); err != nil {
		utilruntime.HandleError(err)
		klog.Error(err, fmt.Sprintf("Error when maintain role for AppService[%s]", apiService.Name))
		return err
	}

	return nil
}

func (c *ApiServiceController) maintainDeployment(target *apis.AppService) error {
	for clusterId, tool := range c.clusterToolMap {
		deploymentName := c.getDeploymentName(target)
		current, err := tool.GetDeployment(target.Namespace, deploymentName)
		replicas := c.replicaTool.GetReplicas(clusterId, target.Spec.TotalReplicas, target.Spec.ReplicaPolicy)
		targetDeployment := c.newDeployment(target, &replicas)
		// Once err occurs during maintaining deployment, the err will be returned and the maintainance will be interrupted
		if err != nil {
			// If the deployment does not exist
			if errors.IsNotFound(err) {
				err := tool.CreateDeployment(targetDeployment)
				if err != nil {
					return err
				}
				klog.Infof("Create deployment in cluster[%s] for AppService[%s]", clusterId, target.Name)
			} else {
				return err
			}
			// If current and target is different, update current.
		} else if c.isDeploymentDifferent(targetDeployment, current) {
			err := tool.UpdateDeployment(targetDeployment, current)
			if err != nil {
				return err
			}
			klog.Infof("Update deployment in cluster[%s] for AppService[%s]", clusterId, target.Name)
		}
	}
	return nil
}

func (c *ApiServiceController) maintainService(target *apis.AppService) error {
	for clusterId, tool := range c.clusterToolMap {
		svcName := c.getServiceName(target)
		current, err := tool.GetService(target.Namespace, svcName)
		targetSvc := c.newService(target)
		if err != nil {
			if errors.IsNotFound(err) {
				err := tool.CreateService(targetSvc)
				if err != nil {
					return err
				}
				klog.Infof("Create service in cluster[%s] for AppService[%s]", clusterId, target.Name)
			} else {
				return err
			}
		} else if c.isSvcDifferent(targetSvc, current) {
			err := tool.UpdateService(c.getUpdatedService(targetSvc, current))
			if err != nil {
				return err
			}
			klog.Infof("Update service in cluster[%s] for AppService[%s]", clusterId, target.Name)
		}
	}
	return nil
}

func (c *ApiServiceController) maintainRole(target *apis.AppService) error {
	// this means there is no Role Definition, return directly
	if len(target.Spec.RoleTemplate.Kind) == 0 {
		return nil
	}
	for clusterId, tool := range c.clusterToolMap {
		if target.Spec.RoleTemplate.Kind == RESOURCE_KIND_ROLE {
			targetRole := c.newRole(target)
			current, err := tool.GetRole(targetRole.Namespace, targetRole.Name)
			if err != nil {
				if errors.IsNotFound(err) {
					err := tool.CreateRole(targetRole)
					if err != nil {
						return nil
					}
					klog.Infof("Create Role[%s] in cluster[%s] for AppService[%s]", targetRole.Namespace, clusterId, target.Name)
				} else {
					return err
				}
			} else if c.isRoleDifferent(targetRole, current) {
				err := tool.UpdateRole(targetRole)
				if err != nil {
					return err
				}
				klog.Infof("Update Role[%s] in cluster[%s] for AppService[%s]", targetRole.Name, clusterId, target.Name)
			}
		} else if target.Spec.RoleTemplate.Kind == RESOURCE_KIND_CLUSTER_ROLE {

		}
	}

	return nil
}

func (c *ApiServiceController) verifyRoleSpec(target *apis.AppService) error {
	kind := target.Spec.RoleTemplate.Kind
	if kind != RESOURCE_KIND_ROLE || kind != RESOURCE_KIND_CLUSTER_ROLE {
		return fmt.Errorf("Spec for RoleKind is not correct. Value: %s", kind)
	}
	return nil
}

func (c *ApiServiceController) newRole(target *apis.AppService) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      target.Spec.RoleTemplate.Metadata.Name,
			Namespace: target.Namespace,
		},
		Rules: target.Spec.RoleTemplate.Rules,
	}
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

func (c *ApiServiceController) newService(appService *apis.AppService) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.getServiceName(appService),
			Namespace: appService.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(appService, apis.SchemeGroupVersion.WithKind(APP_SERVICE_KIND)),
			},
		},
		Spec: appService.Spec.ServiceSpec,
	}
	return svc
}

func (c *ApiServiceController) getUpdatedService(target, current *corev1.Service) *corev1.Service {
	current.ObjectMeta.Annotations = target.ObjectMeta.GetAnnotations()
	current.ObjectMeta.Labels = target.ObjectMeta.GetLabels()
	current.Spec.Type = target.Spec.Type
	current.Spec.Ports = target.Spec.Ports
	current.Spec.LoadBalancerIP = target.Spec.LoadBalancerIP
	current.Spec.Selector = target.Spec.Selector
	return current
}

func (c *ApiServiceController) getUpdatedRole(target, current *rbacv1.Role) *rbacv1.Role {
	current.Rules = target.Rules
	return current
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

func (c *ApiServiceController) isSvcDifferent(target, current *corev1.Service) bool {
	if !reflect.DeepEqual(target.ObjectMeta.Annotations, current.ObjectMeta.Annotations) {
		return true
	}
	if !reflect.DeepEqual(target.ObjectMeta.Labels, current.ObjectMeta.Labels) {
		return true
	}
	if len(target.Spec.Type) > 0 && !reflect.DeepEqual(target.Spec.Type, current.Spec.Type) {
		return true
	}
	if len(target.Spec.LoadBalancerIP) > 0 && !reflect.DeepEqual(target.Spec.LoadBalancerIP, current.Spec.LoadBalancerIP) {
		return true
	}
	if len(target.Spec.Ports) > 0 && !reflect.DeepEqual(target.Spec.Ports, current.Spec.Ports) {
		return true
	}
	if len(target.Spec.Selector) > 0 && !reflect.DeepEqual(target.Spec.Selector, current.Spec.Selector) {
		return true
	}
	return false
}

func (c *ApiServiceController) isRoleDifferent(target, current *rbacv1.Role) bool {
	if !reflect.DeepEqual(target.Rules, current.Rules) {
		return true
	}
	return false
}
