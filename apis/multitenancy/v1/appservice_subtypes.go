package v1

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VirtualServiceSpec struct {
	Hosts []string `json:"hosts,omitempty"`
	// +optional
	Gateways []string                 `json:"gateways,omitempty"`
	Http     []VirtualServiceHttpSpec `json:"http,omitempty"`
}

type VirtualServiceHttpSpec struct {
	// +optional
	Name string `json:"name,omitempty"`
	// +optional
	Match []VirtualServiceUri `json:"match,omitempty"`
	// +optional
	Rewrite VirtualServiceMatch `json:"rewrite,omitempty"`
	// +optional
	Route []VirtualServiceRoute `json:"route,omitempty"`
}

type VirtualServiceMatch struct {
	Uri VirtualServiceUri `json:"uri,omitempty"`
}

type VirtualServiceUri struct {
	Prefix string `json:"prefix,omitempty"`
}

type VirtualServiceRoute struct {
	Route VirtualServiceDestination `json:"route,omitempty"`
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
