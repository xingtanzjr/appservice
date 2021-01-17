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

var APPSERVICE_ROLE_TYPE = "Role"

type RoleReconciler struct {
	AppService     *apis.AppService
	ClusterToolMap map[string]*components.ClusterTool
}

func NewRoleReconciler(appService *apis.AppService, clusterToolMap map[string]*components.ClusterTool) *RoleReconciler {
	return &RoleReconciler{
		AppService:     appService,
		ClusterToolMap: clusterToolMap,
	}
}

func (r *RoleReconciler) GetResource(clusterId string) (runtime.Object, error) {
	kubeClient := r.ClusterToolMap[clusterId].KubeClient
	return kubeClient.RbacV1().Roles(r.AppService.Namespace).Get(context.TODO(), r.GetResourceName(), metav1.GetOptions{})
}

func (r *RoleReconciler) GetResourceName() string {
	return r.AppService.Spec.RoleTemplate.Metadata.Name
}

func (r *RoleReconciler) GetResourceType() string {
	return APPSERVICE_ROLE_TYPE
}

func (r *RoleReconciler) GetClusterToolMap() map[string]*components.ClusterTool {
	return r.ClusterToolMap
}

func (r *RoleReconciler) NewRoleForCreate() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.AppService.Spec.RoleTemplate.Metadata.Name,
			Namespace: r.AppService.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.AppService, apis.SchemeGroupVersion.WithKind(APP_SERVICE_KIND)),
			},
		},
		Rules: r.AppService.Spec.RoleTemplate.Rules,
	}
}

func (r *RoleReconciler) CreateResource(clusterId string) error {
	kubeClient := r.ClusterToolMap[clusterId].KubeClient
	role := r.NewRoleForCreate()
	_, err := kubeClient.RbacV1().Roles(role.Namespace).Create(context.TODO(), role, metav1.CreateOptions{})
	return err
}

func (r *RoleReconciler) IsResourceNeedUpdate(clusterId string) bool {
	target := r.NewRoleForCreate()
	currentObject, _ := r.GetResource(clusterId)
	current := currentObject.(*rbacv1.Role)
	if !reflect.DeepEqual(target.Rules, current.Rules) {
		return true
	}
	return false

}

func (r *RoleReconciler) UpdateResource(clusterId string) error {
	target := r.NewRoleForCreate()
	currentObject, _ := r.GetResource(clusterId)
	current := currentObject.(*rbacv1.Role)
	updatedRole := r.getUpdatedRole(target, current)

	kubeClient := r.ClusterToolMap[clusterId].KubeClient
	_, err := kubeClient.RbacV1().Roles(target.Namespace).Update(context.TODO(), updatedRole, metav1.UpdateOptions{})
	return err
}

func (r *RoleReconciler) getUpdatedRole(target, current *rbacv1.Role) *rbacv1.Role {
	current.Rules = target.Rules
	return current
}
