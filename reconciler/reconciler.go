package reconciler

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	components "metricsadvisor.ai/appservice/components"
)

type Reconciler interface {
	GetResource(clusterId string) (runtime.Object, error)
	CreateResource(clusterId string) error
	IsResourceNeedUpdate(clusterId string) bool
	UpdateResource(clusterId string) error

	GetResourceName() string
	GetResourceType() string
	GetClusterToolMap() map[string]*components.ClusterTool
}

func Reconcile(reconciler Reconciler) error {
	for clusterId := range reconciler.GetClusterToolMap() {
		_, err := reconciler.GetResource(clusterId)
		// Once err occurs during maintaining resource, the err will be returned and the maintainance will be interrupted
		if err != nil {
			// If the resource does not exist, then create it in the cluster
			if errors.IsNotFound(err) {
				err := reconciler.CreateResource(clusterId)
				if err != nil {
					return err
				}
				klog.Infof("Create [%s][%s] in cluster[%s]", reconciler.GetResourceType(), reconciler.GetResourceName(), clusterId)
			} else {
				klog.Errorf("Get resource [%s][%s] error in cluster[%s]", reconciler.GetResourceType(), reconciler.GetResourceName(), clusterId)
				return err
			}
			// If current and target is different, update current.
		} else if reconciler.IsResourceNeedUpdate(clusterId) {
			err := reconciler.UpdateResource(clusterId)
			if err != nil {
				klog.Errorf("Failed to update resource [%s][%s] in cluster[%s]", reconciler.GetResourceType(), reconciler.GetResourceName(), clusterId)
				return err
			}
			klog.Infof("Update resource [%s][%s] in cluster[%s]", reconciler.GetResourceType(), reconciler.GetResourceName(), clusterId)
		}
	}
	return nil
}

func test() {

	r := &DeploymentReconciler{}
	Reconcile(r)
}
