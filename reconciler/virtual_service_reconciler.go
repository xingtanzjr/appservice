package reconciler

import (
	"context"
	"fmt"

	v1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apis "metricsadvisor.ai/appservice/apis/multitenancy/v1"
	components "metricsadvisor.ai/appservice/components"
)

var VIRTUAL_SERVICE_NAME_TEMPLATE = "vs-%s-svc"
var APP_SERVICE_VIRTUAL_SERVICE_TYEP = "VirtualService"

type VirtualServiceReconciler struct {
	AppService     *apis.AppService
	ClusterToolMap map[string]*components.ClusterTool
}

func (r *VirtualServiceReconciler) GetResourceName() string {
	return fmt.Sprintf(VIRTUAL_SERVICE_NAME_TEMPLATE, r.AppService.Name)
}

func (r *VirtualServiceReconciler) GetResourceType() string {
	return APP_SERVICE_VIRTUAL_SERVICE_TYEP
}

func (r *VirtualServiceReconciler) GetClusterToolMap() map[string]*components.ClusterTool {
	return r.ClusterToolMap
}

func (r *VirtualServiceReconciler) GetResource(clusterId string) (runtime.Object, error) {
	istioClient := r.ClusterToolMap[clusterId].IstioClient
	return istioClient.NetworkingV1alpha3().VirtualServices(r.AppService.Namespace).Get(context.TODO(), r.GetResourceName(), metav1.GetOptions{})
}

func (r *VirtualServiceReconciler) newVirtualService() *v1alpha3.VirtualService {
	virtualService := &v1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.GetResourceName(),
			Namespace: r.AppService.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.AppService, apis.SchemeGroupVersion.WithKind(APP_SERVICE_KIND)),
			},
		},
		// Spec: networkingv1alpha3.VirtualService{

		// }

	}
	return virtualService
}

// 	CreateResource(clusterId string) error
// 	IsResourceNeedUpdate(clusterId string) bool
// 	UpdateResource(clusterId string) error
