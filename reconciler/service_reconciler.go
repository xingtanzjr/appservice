package reconciler

import (
	"context"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apis "metricsadvisor.ai/appservice/apis/multitenancy/v1"
	components "metricsadvisor.ai/appservice/components"
)

var APPSERVICE_SERVICE_NAME_SUFFIX = "-svc"
var APPSERVICE_SERVICE_TYPE = "Service"

type ServiceReconciler struct {
	AppService     *apis.AppService
	ClusterToolMap map[string]*components.ClusterTool
}

func NewServiceReconciler(appService *apis.AppService, clusterToolMap map[string]*components.ClusterTool) *ServiceReconciler {
	return &ServiceReconciler{
		AppService:     appService,
		ClusterToolMap: clusterToolMap,
	}
}

func (r *ServiceReconciler) GetResource(clusterId string) (runtime.Object, error) {
	kubeClient := r.ClusterToolMap[clusterId].KubeClient
	return kubeClient.CoreV1().Services(r.AppService.Namespace).Get(context.TODO(), r.GetResourceName(), metav1.GetOptions{})
}

func (r *ServiceReconciler) GetResourceName() string {
	return r.AppService.Name + APPSERVICE_SERVICE_NAME_SUFFIX
}

func (r *ServiceReconciler) GetResourceType() string {
	return APPSERVICE_SERVICE_TYPE
}

func (r *ServiceReconciler) GetClusterToolMap() map[string]*components.ClusterTool {
	return r.ClusterToolMap
}

func (r *ServiceReconciler) NewServiceForCreate() *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.GetResourceName(),
			Namespace: r.AppService.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.AppService, apis.SchemeGroupVersion.WithKind(APP_SERVICE_KIND)),
			},
		},
		Spec: r.AppService.Spec.ServiceSpec,
	}
	return svc
}

func (r *ServiceReconciler) CreateResource(clusterId string) error {
	kubeClient := r.ClusterToolMap[clusterId].KubeClient
	svc := r.NewServiceForCreate()
	_, err := kubeClient.CoreV1().Services(r.AppService.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
	return err
}

func (r *ServiceReconciler) IsResourceNeedUpdate(clusterId string) bool {
	target := r.NewServiceForCreate()
	currentObject, _ := r.GetResource(clusterId)
	current := currentObject.(*corev1.Service)
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

func (r *ServiceReconciler) UpdateResource(clusterId string) error {
	target := r.NewServiceForCreate()
	currentObject, _ := r.GetResource(clusterId)
	current := currentObject.(*corev1.Service)
	updatedSvc := r.getUpdatedService(target, current)

	kubeClient := r.ClusterToolMap[clusterId].KubeClient
	_, err := kubeClient.CoreV1().Services(target.Namespace).Update(context.TODO(), updatedSvc, metav1.UpdateOptions{})
	return err
}

func (r *ServiceReconciler) getUpdatedService(target, current *corev1.Service) *corev1.Service {
	current.ObjectMeta.Annotations = target.ObjectMeta.GetAnnotations()
	current.ObjectMeta.Labels = target.ObjectMeta.GetLabels()
	current.Spec.Type = target.Spec.Type
	current.Spec.Ports = target.Spec.Ports
	current.Spec.LoadBalancerIP = target.Spec.LoadBalancerIP
	current.Spec.Selector = target.Spec.Selector
	return current
}
