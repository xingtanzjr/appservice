package components

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
)

var APP_SERVICE_WORK_QUEUE_NAME = "ApiServiceWorkQueue"
var APP_SERVICE_DEPLOYMENT_SUFFIX = "-deployment"
var APP_SERVICE_KIND = "AppService"

type EventItem struct {
	ClusterId string
	Key       string
}

func (item *EventItem) String() string {
	return fmt.Sprintf("Cluster[%s]-[%s]", item.ClusterId, item.Key)
}

type ApiServiceController struct {
	clusterToolMap map[string]*ClusterTool
	workqueue      workqueue.RateLimitingInterface
	replicaTool    ReplicaTool
	//TODO learn more about Recorder and leverage it in current program
}

func NewApiServiceController(clusterToolMap map[string]*ClusterTool) *ApiServiceController {
	controller := &ApiServiceController{
		clusterToolMap: clusterToolMap,
		workqueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), APP_SERVICE_WORK_QUEUE_NAME),
		replicaTool:    NewReplicaTool(&clusterToolMap),
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

	// Create sub resources in each cluster

	// Create deployment
	if err := c.maintainDeployment(apiService); err != nil {
		utilruntime.HandleError(err)
		klog.Error(err, fmt.Sprintf("Error when maintain deployments for app service[%s]", apiService.Name))
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
		} else if !reflect.DeepEqual(targetDeployment.Spec, current.Spec) {
			err := tool.UpdateDeployment(targetDeployment, current)
			if err != nil {
				return err
			}
			klog.Infof("Update deployment in cluster[%s] for AppService[%s]", clusterId, target.Name)
		}
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
	updateTimeInCluster := GetResourceUpdateTime(current.ObjectMeta)
	updateTimeOfTarget := GetResourceUpdateTime(target.ObjectMeta)
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
