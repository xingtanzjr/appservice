package components

import (
	"context"
	"reflect"

	"k8s.io/client-go/kubernetes"
	appsListers "k8s.io/client-go/listers/apps/v1"
	coreLister "k8s.io/client-go/listers/core/v1"
	rbacLister "k8s.io/client-go/listers/rbac/v1"
	asClient "metricsadvisor.ai/appservice/generated/multitenancy/clientset/versioned"
	asInformer "metricsadvisor.ai/appservice/generated/multitenancy/informers/externalversions/multitenancy/v1"
	asLister "metricsadvisor.ai/appservice/generated/multitenancy/listers/multitenancy/v1"

	istioClient "istio.io/client-go/pkg/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apis "metricsadvisor.ai/appservice/apis/multitenancy/v1"
)

type ClusterTool struct {
	ClusterId string

	// Scalfold for ApiService
	ApiServiceLister   asLister.AppServiceLister
	ApiServiceClient   *asClient.Clientset
	ApiServiceInformer asInformer.AppServiceInformer

	// Scalfold for basic kubernetes resources
	DeploymentLister         appsListers.DeploymentLister
	ServiceLister            coreLister.ServiceLister
	ServiceAccountLister     coreLister.ServiceAccountLister
	RoleLister               rbacLister.RoleLister
	RoleBindingLister        rbacLister.RoleBindingLister
	ClusterRoleLister        rbacLister.ClusterRoleLister
	ClusterRoleBindingLister rbacLister.ClusterRoleBindingLister
	KubeClient               *kubernetes.Clientset

	// Istio scalfold
	IstioClient *istioClient.Clientset
}

func (c *ClusterTool) GetApiService(namespace string, name string) (*apis.AppService, error) {
	return c.ApiServiceLister.AppServices(namespace).Get(name)
}

func (c *ClusterTool) CreateApiService(apiService *apis.AppService) error {
	_, err := c.ApiServiceClient.MultitenancyV1().AppServices(apiService.Namespace).Create(context.TODO(), c.NewApiServiceForCreate(apiService), metav1.CreateOptions{})
	return err
}

func (c *ClusterTool) UpdateApiService(old, new *apis.AppService) error {
	old.Spec = *new.Spec.DeepCopy()
	_, err := c.ApiServiceClient.MultitenancyV1().AppServices(old.Namespace).Update(context.TODO(), old, metav1.UpdateOptions{})
	return err
}

func (c *ClusterTool) NewApiServiceForCreate(apiService *apis.AppService) *apis.AppService {
	return &apis.AppService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiService.ObjectMeta.Name,
			Namespace: apiService.ObjectMeta.Namespace,
		},
		Spec: *apiService.Spec.DeepCopy(),
	}
}

func (c *ClusterTool) IsApiServiceDifferent(as1, as2 *apis.AppService) bool {
	if !reflect.DeepEqual(as1.Spec, as2.Spec) {
		return true
	}
	return false
}

func (c *ClusterTool) GetDeployment(namespace, name string) (*appsv1.Deployment, error) {
	return c.DeploymentLister.Deployments(namespace).Get(name)
}

func (c *ClusterTool) CreateDeployment(deployment *appsv1.Deployment) error {
	_, err := c.KubeClient.AppsV1().Deployments(deployment.Namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
	return err
}

func (c *ClusterTool) UpdateDeployment(target, current *appsv1.Deployment) error {
	current.Spec = *target.Spec.DeepCopy()
	_, err := c.KubeClient.AppsV1().Deployments(current.Namespace).Update(context.TODO(), current, metav1.UpdateOptions{})
	return err
}

func (c *ClusterTool) GetService(namespace, name string) (*corev1.Service, error) {
	return c.KubeClient.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (c *ClusterTool) CreateService(svc *corev1.Service) error {
	_, err := c.KubeClient.CoreV1().Services(svc.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
	return err
}

func (c *ClusterTool) UpdateService(target *corev1.Service) error {
	_, err := c.KubeClient.CoreV1().Services(target.Namespace).Update(context.TODO(), target, metav1.UpdateOptions{})
	return err
}

func GetResourceUpdateTime(meta metav1.ObjectMeta) *metav1.Time {
	if meta.ManagedFields == nil || len(meta.ManagedFields) == 0 {
		return nil
	}

	var ret = meta.ManagedFields[0].Time
	for _, entity := range meta.ManagedFields {
		if ret == nil || ret.Before(entity.Time) {
			ret = entity.Time
		}
	}
	return ret
}
