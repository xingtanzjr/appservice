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

var APPSERVICE_CLUSTERROLE_TYPE = "ClusterRole"

type ClusterRoleReconciler struct {
	AppService     *apis.AppService
	ClusterToolMap map[string]*components.ClusterTool
}

func NewClusterRoleReconciler(appService *apis.AppService, clusterToolMap map[string]*components.ClusterTool) *ClusterRoleReconciler {
	return &ClusterRoleReconciler{
		AppService:     appService,
		ClusterToolMap: clusterToolMap,
	}
}

func (r *ClusterRoleReconciler) GetResource(clusterId string) (runtime.Object, error) {
	kubeClient := r.ClusterToolMap[clusterId].KubeClient
	return kubeClient.RbacV1().ClusterRoles().Get(context.TODO(), r.GetResourceName(), metav1.GetOptions{})
}

func (r *ClusterRoleReconciler) GetResourceName() string {
	return r.AppService.Spec.RoleTemplate.Metadata.Name
}

func (r *ClusterRoleReconciler) GetResourceType() string {
	return APPSERVICE_CLUSTERROLE_TYPE
}

func (r *ClusterRoleReconciler) GetClusterToolMap() map[string]*components.ClusterTool {
	return r.ClusterToolMap
}

func (r *ClusterRoleReconciler) newClusterRoleForCreate() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.AppService.Spec.RoleTemplate.Metadata.Name,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.AppService, apis.SchemeGroupVersion.WithKind(APP_SERVICE_KIND)),
			},
		},
		Rules: r.AppService.Spec.RoleTemplate.Rules,
	}
}

func (r *ClusterRoleReconciler) CreateResource(clusterId string) error {
	kubeClient := r.ClusterToolMap[clusterId].KubeClient
	role := r.newClusterRoleForCreate()
	_, err := kubeClient.RbacV1().ClusterRoles().Create(context.TODO(), role, metav1.CreateOptions{})
	return err
}

func (r *ClusterRoleReconciler) IsResourceNeedUpdate(clusterId string) bool {
	target := r.newClusterRoleForCreate()
	currentObject, _ := r.GetResource(clusterId)
	current := currentObject.(*rbacv1.ClusterRole)
	if !reflect.DeepEqual(target.Rules, current.Rules) {
		return true
	}
	return false

}

func (r *ClusterRoleReconciler) UpdateResource(clusterId string) error {
	target := r.newClusterRoleForCreate()
	currentObject, _ := r.GetResource(clusterId)
	current := currentObject.(*rbacv1.ClusterRole)
	updatedRole := r.getUpdatedClusterRole(target, current)

	kubeClient := r.ClusterToolMap[clusterId].KubeClient
	_, err := kubeClient.RbacV1().ClusterRoles().Update(context.TODO(), updatedRole, metav1.UpdateOptions{})
	return err
}

func (r *ClusterRoleReconciler) getUpdatedClusterRole(target, current *rbacv1.ClusterRole) *rbacv1.ClusterRole {
	current.Rules = target.Rules
	return current
}
