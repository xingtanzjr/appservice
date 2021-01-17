package reconciler

import (
	"context"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apis "metricsadvisor.ai/appservice/apis/multitenancy/v1"
	components "metricsadvisor.ai/appservice/components"
)

var APP_SERVICE_KIND = "AppService"
var APP_SERVICE_DEPLOYMENT_SUFFIX = "-deployment"
var APP_SERVICE_DEPLOYMENT_TYPE = "Deployment"

type DeploymentReconciler struct {
	AppService     apis.AppService
	ClusterToolMap map[string]*components.ClusterTool
	replicaTool    ReplicaTool
}

func NewDeploymentReconciler(appService apis.AppService, clusterToolMap map[string]*components.ClusterTool) *DeploymentReconciler {
	replicaTool := NewReplicaTool(&clusterToolMap)
	return &DeploymentReconciler{
		AppService:     appService,
		ClusterToolMap: clusterToolMap,
		replicaTool:    replicaTool,
	}
}

func (r *DeploymentReconciler) GetResourceName() string {
	return r.AppService.Name + APP_SERVICE_DEPLOYMENT_SUFFIX
}

func (r *DeploymentReconciler) GetResourceType() string {
	return APP_SERVICE_DEPLOYMENT_TYPE
}

func (r *DeploymentReconciler) GetClusterToolMap() map[string]*components.ClusterTool {
	return r.ClusterToolMap
}

func (r *DeploymentReconciler) IsResourceNeedUpdate(clusterId string) bool {
	target := r.NewResourceForCreate(clusterId).(*appsv1.Deployment)
	currentObject, _ := r.GetResource(clusterId)
	current := currentObject.(*appsv1.Deployment)
	if *target.Spec.Replicas != *current.Spec.Replicas {
		return true
	}
	if !reflect.DeepEqual(target.Spec.Selector, current.Spec.Selector) {
		return true
	}
	if !reflect.DeepEqual(target.Spec.Template, current.Spec.Template) {
		return true
	}
	return false
}

func (r *DeploymentReconciler) GetResource(clusterId string) (runtime.Object, error) {
	return r.ClusterToolMap[clusterId].GetDeployment(r.AppService.Namespace, r.GetResourceName())
}

func (r *DeploymentReconciler) CreateResource(clusterId string) error {
	return r.ClusterToolMap[clusterId].CreateDeployment(r.NewResourceForCreate(clusterId).(*appsv1.Deployment))
}

func (r *DeploymentReconciler) UpdateResource(clusterId string) error {
	target := r.NewResourceForCreate(clusterId).(*appsv1.Deployment)
	currentObject, _ := r.GetResource(clusterId)
	current := currentObject.(*appsv1.Deployment)

	current.Spec = *target.Spec.DeepCopy()
	kubeClient := r.ClusterToolMap[clusterId].KubeClient
	_, err := kubeClient.AppsV1().Deployments(current.Namespace).Update(context.TODO(), current, metav1.UpdateOptions{})
	return err
}

func (r *DeploymentReconciler) NewResourceForCreate(clusterId string) runtime.Object {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.GetResourceName(),
			Namespace: r.AppService.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(&r.AppService, apis.SchemeGroupVersion.WithKind(APP_SERVICE_KIND)),
			},
		},
		Spec: r.AppService.Spec.DeploymentSpec,
	}
	replicas := r.replicaTool.GetReplicas(clusterId, r.AppService.Spec.TotalReplicas, r.AppService.Spec.ReplicaPolicy)
	deployment.Spec.Replicas = &replicas
	return deployment
}

func (r *DeploymentReconciler) newDeployment(clusterId string) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.GetResourceName(),
			Namespace: r.AppService.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(&r.AppService, apis.SchemeGroupVersion.WithKind(APP_SERVICE_KIND)),
			},
		},
		Spec: r.AppService.Spec.DeploymentSpec,
	}
	replicas := r.replicaTool.GetReplicas(clusterId, r.AppService.Spec.TotalReplicas, r.AppService.Spec.ReplicaPolicy)
	deployment.Spec.Replicas = &replicas
	return deployment
}
