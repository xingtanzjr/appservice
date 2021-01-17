package reconciler

import (
	"context"
	"fmt"
	"reflect"

	networkingv1alpha3 "istio.io/api/networking/v1alpha3"
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

func NewVirtualServiceReconciler(appService *apis.AppService, clusterToolMap map[string]*components.ClusterTool) *VirtualServiceReconciler {
	return &VirtualServiceReconciler{
		AppService:     appService,
		ClusterToolMap: clusterToolMap,
	}
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
	//TODO: add validation here
	rawSpec := r.AppService.Spec.VirtualServiceSpec
	spec := networkingv1alpha3.VirtualService{
		Hosts: rawSpec.Hosts,
		Http: []*networkingv1alpha3.HTTPRoute{
			{
				Name: rawSpec.Http[0].Name,
				Match: []*networkingv1alpha3.HTTPMatchRequest{
					{
						Uri: &networkingv1alpha3.StringMatch{
							MatchType: &networkingv1alpha3.StringMatch_Prefix{
								Prefix: rawSpec.Http[0].Match[0].Uri.Prefix,
							},
						},
					},
				},
				Rewrite: &networkingv1alpha3.HTTPRewrite{
					Uri: rawSpec.Http[0].Rewrite.Uri,
				},
				Route: []*networkingv1alpha3.HTTPRouteDestination{
					{
						Destination: &networkingv1alpha3.Destination{
							Host: rawSpec.Http[0].Route[0].Destination.Host,
						},
					},
				},
			},
		},
	}

	virtualService := &v1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.GetResourceName(),
			Namespace: r.AppService.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.AppService, apis.SchemeGroupVersion.WithKind(APP_SERVICE_KIND)),
			},
		},
		Spec: spec,
	}
	return virtualService
}

func (r *VirtualServiceReconciler) CreateResource(clusterId string) error {
	istioClient := r.ClusterToolMap[clusterId].IstioClient
	_, err := istioClient.NetworkingV1alpha3().VirtualServices(r.AppService.Namespace).Create(context.TODO(), r.newVirtualService(), metav1.CreateOptions{})
	return err
}

func (r *VirtualServiceReconciler) IsResourceNeedUpdate(clusterId string) bool {
	currentObject, _ := r.GetResource(clusterId)
	current := currentObject.(*v1alpha3.VirtualService)
	target := r.newVirtualService()
	if !reflect.DeepEqual(target.Spec, current.Spec) {
		return true
	}
	return false
}

func (r *VirtualServiceReconciler) UpdateResource(clusterId string) error {
	currentObject, _ := r.GetResource(clusterId)
	current := currentObject.(*v1alpha3.VirtualService)
	target := r.newVirtualService()
	current.Spec = target.Spec
	istioClient := r.ClusterToolMap[clusterId].IstioClient
	_, err := istioClient.NetworkingV1alpha3().VirtualServices(r.AppService.Namespace).Update(context.TODO(), current, metav1.UpdateOptions{})
	return err
}
