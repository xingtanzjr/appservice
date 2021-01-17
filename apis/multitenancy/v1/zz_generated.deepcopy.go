// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	rbacv1 "k8s.io/api/rbac/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AppService) DeepCopyInto(out *AppService) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AppService.
func (in *AppService) DeepCopy() *AppService {
	if in == nil {
		return nil
	}
	out := new(AppService)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AppService) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AppServiceList) DeepCopyInto(out *AppServiceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AppService, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AppServiceList.
func (in *AppServiceList) DeepCopy() *AppServiceList {
	if in == nil {
		return nil
	}
	out := new(AppServiceList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AppServiceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AppServiceSpec) DeepCopyInto(out *AppServiceSpec) {
	*out = *in
	if in.TotalReplicas != nil {
		in, out := &in.TotalReplicas, &out.TotalReplicas
		*out = new(int32)
		**out = **in
	}
	in.DeploymentSpec.DeepCopyInto(&out.DeploymentSpec)
	in.ServiceSpec.DeepCopyInto(&out.ServiceSpec)
	in.PodAutoScalerSpec.DeepCopyInto(&out.PodAutoScalerSpec)
	in.RoleTemplate.DeepCopyInto(&out.RoleTemplate)
	in.RoleBindingTemplate.DeepCopyInto(&out.RoleBindingTemplate)
	in.ServiceAccountMeta.DeepCopyInto(&out.ServiceAccountMeta)
	in.VirtualServiceSpec.DeepCopyInto(&out.VirtualServiceSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AppServiceSpec.
func (in *AppServiceSpec) DeepCopy() *AppServiceSpec {
	if in == nil {
		return nil
	}
	out := new(AppServiceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AppServiceStatus) DeepCopyInto(out *AppServiceStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AppServiceStatus.
func (in *AppServiceStatus) DeepCopy() *AppServiceStatus {
	if in == nil {
		return nil
	}
	out := new(AppServiceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RoleBindingTemplate) DeepCopyInto(out *RoleBindingTemplate) {
	*out = *in
	in.Metadata.DeepCopyInto(&out.Metadata)
	if in.Subjects != nil {
		in, out := &in.Subjects, &out.Subjects
		*out = make([]rbacv1.Subject, len(*in))
		copy(*out, *in)
	}
	out.RoleRef = in.RoleRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RoleBindingTemplate.
func (in *RoleBindingTemplate) DeepCopy() *RoleBindingTemplate {
	if in == nil {
		return nil
	}
	out := new(RoleBindingTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RoleTemplate) DeepCopyInto(out *RoleTemplate) {
	*out = *in
	in.Metadata.DeepCopyInto(&out.Metadata)
	if in.Rules != nil {
		in, out := &in.Rules, &out.Rules
		*out = make([]rbacv1.PolicyRule, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RoleTemplate.
func (in *RoleTemplate) DeepCopy() *RoleTemplate {
	if in == nil {
		return nil
	}
	out := new(RoleTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualServiceDestination) DeepCopyInto(out *VirtualServiceDestination) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualServiceDestination.
func (in *VirtualServiceDestination) DeepCopy() *VirtualServiceDestination {
	if in == nil {
		return nil
	}
	out := new(VirtualServiceDestination)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualServiceHttpSpec) DeepCopyInto(out *VirtualServiceHttpSpec) {
	*out = *in
	if in.Match != nil {
		in, out := &in.Match, &out.Match
		*out = make([]VirtualServiceMatch, len(*in))
		copy(*out, *in)
	}
	out.Rewrite = in.Rewrite
	if in.Route != nil {
		in, out := &in.Route, &out.Route
		*out = make([]VirtualServiceRoute, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualServiceHttpSpec.
func (in *VirtualServiceHttpSpec) DeepCopy() *VirtualServiceHttpSpec {
	if in == nil {
		return nil
	}
	out := new(VirtualServiceHttpSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualServiceMatch) DeepCopyInto(out *VirtualServiceMatch) {
	*out = *in
	out.Uri = in.Uri
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualServiceMatch.
func (in *VirtualServiceMatch) DeepCopy() *VirtualServiceMatch {
	if in == nil {
		return nil
	}
	out := new(VirtualServiceMatch)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualServiceRewrite) DeepCopyInto(out *VirtualServiceRewrite) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualServiceRewrite.
func (in *VirtualServiceRewrite) DeepCopy() *VirtualServiceRewrite {
	if in == nil {
		return nil
	}
	out := new(VirtualServiceRewrite)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualServiceRoute) DeepCopyInto(out *VirtualServiceRoute) {
	*out = *in
	out.Destination = in.Destination
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualServiceRoute.
func (in *VirtualServiceRoute) DeepCopy() *VirtualServiceRoute {
	if in == nil {
		return nil
	}
	out := new(VirtualServiceRoute)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualServiceSpec) DeepCopyInto(out *VirtualServiceSpec) {
	*out = *in
	if in.Hosts != nil {
		in, out := &in.Hosts, &out.Hosts
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Gateways != nil {
		in, out := &in.Gateways, &out.Gateways
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Http != nil {
		in, out := &in.Http, &out.Http
		*out = make([]VirtualServiceHttpSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualServiceSpec.
func (in *VirtualServiceSpec) DeepCopy() *VirtualServiceSpec {
	if in == nil {
		return nil
	}
	out := new(VirtualServiceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualServiceUri) DeepCopyInto(out *VirtualServiceUri) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualServiceUri.
func (in *VirtualServiceUri) DeepCopy() *VirtualServiceUri {
	if in == nil {
		return nil
	}
	out := new(VirtualServiceUri)
	in.DeepCopyInto(out)
	return out
}
