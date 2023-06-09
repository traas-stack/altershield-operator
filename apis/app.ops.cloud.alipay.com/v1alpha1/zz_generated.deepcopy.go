//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2023.

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

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ChangePod) DeepCopyInto(out *ChangePod) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ChangePod.
func (in *ChangePod) DeepCopy() *ChangePod {
	if in == nil {
		return nil
	}
	out := new(ChangePod)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ChangePod) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ChangePodList) DeepCopyInto(out *ChangePodList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ChangePod, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ChangePodList.
func (in *ChangePodList) DeepCopy() *ChangePodList {
	if in == nil {
		return nil
	}
	out := new(ChangePodList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ChangePodList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ChangePodSpec) DeepCopyInto(out *ChangePodSpec) {
	*out = *in
	if in.PodInfos != nil {
		in, out := &in.PodInfos, &out.PodInfos
		*out = make([]PodSummary, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ChangePodSpec.
func (in *ChangePodSpec) DeepCopy() *ChangePodSpec {
	if in == nil {
		return nil
	}
	out := new(ChangePodSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ChangePodStatus) DeepCopyInto(out *ChangePodStatus) {
	*out = *in
	if in.PodResults != nil {
		in, out := &in.PodResults, &out.PodResults
		*out = make([]PodSummary, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ChangePodStatus.
func (in *ChangePodStatus) DeepCopy() *ChangePodStatus {
	if in == nil {
		return nil
	}
	out := new(ChangePodStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ChangeWorkload) DeepCopyInto(out *ChangeWorkload) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ChangeWorkload.
func (in *ChangeWorkload) DeepCopy() *ChangeWorkload {
	if in == nil {
		return nil
	}
	out := new(ChangeWorkload)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ChangeWorkload) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ChangeWorkloadList) DeepCopyInto(out *ChangeWorkloadList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ChangeWorkload, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ChangeWorkloadList.
func (in *ChangeWorkloadList) DeepCopy() *ChangeWorkloadList {
	if in == nil {
		return nil
	}
	out := new(ChangeWorkloadList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ChangeWorkloadList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ChangeWorkloadSpec) DeepCopyInto(out *ChangeWorkloadSpec) {
	*out = *in
	if in.Policies != nil {
		in, out := &in.Policies, &out.Policies
		*out = make([]DefensePolicy, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ChangeWorkloadSpec.
func (in *ChangeWorkloadSpec) DeepCopy() *ChangeWorkloadSpec {
	if in == nil {
		return nil
	}
	out := new(ChangeWorkloadSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ChangeWorkloadStatus) DeepCopyInto(out *ChangeWorkloadStatus) {
	*out = *in
	if in.DefensePreparingPods != nil {
		in, out := &in.DefensePreparingPods, &out.DefensePreparingPods
		*out = make([]PodSummary, len(*in))
		copy(*out, *in)
	}
	if in.DefenseCheckingPods != nil {
		in, out := &in.DefenseCheckingPods, &out.DefenseCheckingPods
		*out = make([]PodSummary, len(*in))
		copy(*out, *in)
	}
	if in.DefenseCheckPassPods != nil {
		in, out := &in.DefenseCheckPassPods, &out.DefenseCheckPassPods
		*out = make([]PodSummary, len(*in))
		copy(*out, *in)
	}
	if in.DefenseCheckFailPods != nil {
		in, out := &in.DefenseCheckFailPods, &out.DefenseCheckFailPods
		*out = make([]PodSummary, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ChangeWorkloadStatus.
func (in *ChangeWorkloadStatus) DeepCopy() *ChangeWorkloadStatus {
	if in == nil {
		return nil
	}
	out := new(ChangeWorkloadStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DefensePolicy) DeepCopyInto(out *DefensePolicy) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DefensePolicy.
func (in *DefensePolicy) DeepCopy() *DefensePolicy {
	if in == nil {
		return nil
	}
	out := new(DefensePolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OpsConfigInfo) DeepCopyInto(out *OpsConfigInfo) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OpsConfigInfo.
func (in *OpsConfigInfo) DeepCopy() *OpsConfigInfo {
	if in == nil {
		return nil
	}
	out := new(OpsConfigInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OpsConfigInfo) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OpsConfigInfoList) DeepCopyInto(out *OpsConfigInfoList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OpsConfigInfo, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OpsConfigInfoList.
func (in *OpsConfigInfoList) DeepCopy() *OpsConfigInfoList {
	if in == nil {
		return nil
	}
	out := new(OpsConfigInfoList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OpsConfigInfoList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OpsConfigInfoSpec) DeepCopyInto(out *OpsConfigInfoSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OpsConfigInfoSpec.
func (in *OpsConfigInfoSpec) DeepCopy() *OpsConfigInfoSpec {
	if in == nil {
		return nil
	}
	out := new(OpsConfigInfoSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OpsConfigInfoStatus) DeepCopyInto(out *OpsConfigInfoStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OpsConfigInfoStatus.
func (in *OpsConfigInfoStatus) DeepCopy() *OpsConfigInfoStatus {
	if in == nil {
		return nil
	}
	out := new(OpsConfigInfoStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodSummary) DeepCopyInto(out *PodSummary) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodSummary.
func (in *PodSummary) DeepCopy() *PodSummary {
	if in == nil {
		return nil
	}
	out := new(PodSummary)
	in.DeepCopyInto(out)
	return out
}
