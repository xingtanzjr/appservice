package v1

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VirtualServiceSpec struct {
	Hosts    []string                 `json:"hosts,omitempty"`
	Gateways []string                 `json:"gateways,omitempty"`
	Http     []VirtualServiceHttpSpec `json:"http,omitempty"`
}

type VirtualServiceHttpSpec struct {
	Name    string                      `json:"name,omitempty"`
	Match   []VirtualServiceUri         `json:"match,omitempty"`
	Rewrite VirtualServiceUri           `json:"rewrite,omitempty"`
	Route   []VirtualServiceDestination `json:"route,omitempty"`
}

type VirtualServiceUri struct {
	Prefix string `json:"prefix,omitempty"`
}

type VirtualServiceDestination struct {
	Host   string `json:"host,omitempty"`
	Subset string `json:"subset,omitempty"`
}

type RoleTemplate struct {
	// +optional
	Kind string `json:"kind,omitempty"`
	// +optional
	Metadata metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Rules []rbacv1.PolicyRule `json:"rules,omitempty"`
}

type RoleBindingTemplate struct {
	// +optional
	Kind string `json:"kind,omitempty"`
	// +optional
	Metadata metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Subjects []rbacv1.Subject `json:"subjects,omitempty"`
	// +optional
	RoleRef rbacv1.RoleRef `json:"roleRef,omitempty"`
}
