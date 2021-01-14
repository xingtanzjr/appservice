/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AppServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	TotalReplicas  *int32                `json:"totalReplicas"`
	ReplicaPolicy  string                `json:"replicaPolicy"`
	DeploymentSpec appsv1.DeploymentSpec `json:"deploymentSpec"`
	ServiceSpec    corev1.ServiceSpec    `json:"serviceSpec"`
	// +optional
	PodAutoScalerSpec autoscalingv1.HorizontalPodAutoscalerSpec `json:"podAutoScalerSpec,omitempty"`

	// Spec for role
	// +optional
	RoleTemplate RoleTemplate `json:"roleTemplate,omitempty"`

	// Spec for role-bindings
	// +optional
	RoleBindingTemplate RoleBindingTemplate `json:"roleBindingTemplate,omitempty"`

	// Spec for service account
	// +optional
	ServiceAccountMeta metav1.ObjectMeta `json:"serviceAccountMeta,omitempty"`

	// Spec for istio virtual service
	// +optional
	VirtualServiceSpec VirtualServiceSpec `json:"virtualServiceSpec,omitempty"`
}

type AppServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +kubebuilder:object:root=true
type AppService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppServiceSpec   `json:"spec,omitempty"`
	Status AppServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type AppServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AppService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AppService{}, &AppServiceList{})
}
