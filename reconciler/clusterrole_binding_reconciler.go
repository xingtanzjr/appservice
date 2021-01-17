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

var APPSERVICE_CLUSTER_ROLE_BINDING_TYPE = "ClusterRoleBinding"

type ClusterRoleBindingReconciler struct {
	AppService     *apis.AppService
	ClusterToolMap map[string]*components.ClusterTool
}

func NewClusterRoleBindingReconciler(AppService *apis.AppService, ClusterToolMap map[string]*components.ClusterTool) *ClusterRoleBindingReconciler {
	return &ClusterRoleBindingReconciler{
		AppService:     AppService,
		ClusterToolMap: ClusterToolMap,
	}
}

func (r *ClusterRoleBindingReconciler) GetResourceName() string {
	return r.AppService.Spec.RoleBindingTemplate.Metadata.Name
}

func (r *ClusterRoleBindingReconciler) GetResourceType() string {
	return APPSERVICE_CLUSTER_ROLE_BINDING_TYPE
}

func (r *ClusterRoleBindingReconciler) GetClusterToolMap() map[string]*components.ClusterTool {
	return r.ClusterToolMap
}

func (r *ClusterRoleBindingReconciler) GetResource(clusterId string) (runtime.Object, error) {
	kubeClient := r.ClusterToolMap[clusterId].KubeClient
	return kubeClient.RbacV1().ClusterRoleBindings().Get(context.TODO(), r.GetResourceName(), metav1.GetOptions{})
}

func (r *ClusterRoleBindingReconciler) CreateResource(clusterId string) error {
	kubeClient := r.ClusterToolMap[clusterId].KubeClient
	_, err := kubeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), r.newClusterRoleBinding(), metav1.CreateOptions{})
	return err
}

func (r *ClusterRoleBindingReconciler) IsResourceNeedUpdate(clusterId string) bool {
	currentObject, _ := r.GetResource(clusterId)
	current := currentObject.(*rbacv1.ClusterRoleBinding)
	target := r.newClusterRoleBinding()
	if !reflect.DeepEqual(target.Subjects, current.Subjects) {
		return true
	}
	if !reflect.DeepEqual(target.RoleRef, current.RoleRef) {
		return true
	}
	return false
}

func (r *ClusterRoleBindingReconciler) UpdateResource(clusterId string) error {
	target := r.newClusterRoleBinding()
	currentObject, _ := r.GetResource(clusterId)
	current := currentObject.(*rbacv1.ClusterRoleBinding)
	current.Subjects = target.Subjects
	current.RoleRef = target.RoleRef
	kubeClient := r.ClusterToolMap[clusterId].KubeClient
	_, err := kubeClient.RbacV1().ClusterRoleBindings().Update(context.TODO(), current, metav1.UpdateOptions{})
	return err
}

func (r *ClusterRoleBindingReconciler) newClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	rolebinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: r.AppService.Spec.RoleBindingTemplate.Metadata,
		Subjects:   r.AppService.Spec.RoleBindingTemplate.Subjects,
		RoleRef:    r.AppService.Spec.RoleBindingTemplate.RoleRef,
	}
	rolebinding.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(r.AppService, apis.SchemeGroupVersion.WithKind(APP_SERVICE_KIND)),
	}
	return rolebinding
}
