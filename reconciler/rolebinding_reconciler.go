package reconciler

import (
	"context"
	"reflect"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apis "metricsadvisor.ai/appservice/apis/multitenancy/v1"
	components "metricsadvisor.ai/appservice/components"
)

var APPSERVICE_ROLE_BINDING_TYPE = "RoleBinding"

type RoleBindingReconciler struct {
	AppService     *apis.AppService
	ClusterToolMap map[string]*components.ClusterTool
}

func NewRoleBindingReconciler(AppService *apis.AppService, ClusterToolMap map[string]*components.ClusterTool) *RoleBindingReconciler {
	return &RoleBindingReconciler{
		AppService:     AppService,
		ClusterToolMap: ClusterToolMap,
	}
}

func (r *RoleBindingReconciler) GetResourceName() string {
	return r.AppService.Spec.RoleBindingTemplate.Metadata.Name
}

func (r *RoleBindingReconciler) GetResourceType() string {
	return APPSERVICE_ROLE_BINDING_TYPE
}

func (r *RoleBindingReconciler) GetClusterToolMap() map[string]*components.ClusterTool {
	return r.ClusterToolMap
}

func (r *RoleBindingReconciler) GetResource(clusterId string) (runtime.Object, error) {
	kubeClient := r.ClusterToolMap[clusterId].KubeClient
	return kubeClient.RbacV1().RoleBindings(r.AppService.Namespace).Get(context.TODO(), r.GetResourceName(), metav1.GetOptions{})
}

func (r *RoleBindingReconciler) CreateResource(clusterId string) error {
	kubeClient := r.ClusterToolMap[clusterId].KubeClient
	_, err := kubeClient.RbacV1().RoleBindings(r.AppService.Namespace).Create(context.TODO(), r.newRoleBinding(), metav1.CreateOptions{})
	return err
}

func (r *RoleBindingReconciler) IsResourceNeedUpdate(clusterId string) bool {
	currentObject, _ := r.GetResource(clusterId)
	current := currentObject.(*rbacv1.RoleBinding)
	target := r.newRoleBinding()
	if !reflect.DeepEqual(target.Subjects, current.Subjects) {
		return true
	}
	if !reflect.DeepEqual(target.RoleRef, current.RoleRef) {
		return true
	}
	return false
}

func (r *RoleBindingReconciler) UpdateResource(clusterId string) error {
	target := r.newRoleBinding()
	currentObject, _ := r.GetResource(clusterId)
	current := currentObject.(*rbacv1.RoleBinding)
	current.Subjects = target.Subjects
	current.RoleRef = target.RoleRef
	kubeClient := r.ClusterToolMap[clusterId].KubeClient
	_, err := kubeClient.RbacV1().RoleBindings(r.AppService.Namespace).Update(context.TODO(), current, metav1.UpdateOptions{})
	return err
}

func (r *RoleBindingReconciler) newRoleBinding() *rbacv1.RoleBinding {
	rolebinding := &rbacv1.RoleBinding{
		ObjectMeta: r.AppService.Spec.RoleBindingTemplate.Metadata,
		Subjects:   r.AppService.Spec.RoleBindingTemplate.Subjects,
		RoleRef:    r.AppService.Spec.RoleBindingTemplate.RoleRef,
	}
	rolebinding.ObjectMeta.Namespace = r.AppService.Namespace
	rolebinding.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(r.AppService, apis.SchemeGroupVersion.WithKind(APP_SERVICE_KIND)),
	}
	return rolebinding
}
