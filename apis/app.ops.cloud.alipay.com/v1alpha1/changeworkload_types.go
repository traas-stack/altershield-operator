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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	// Init 初始，未调用管控端
	// Init Initially, the control side is not called
	Init = ""
	// Running 运行中，已调用管控端生成发布单，但已发布的数量未达到设置值，还需要继续发布
	// Running, the control side has been called to generate the release order, but the number of published has not reached the set value, and it needs to continue to be published
	Running = "Running"
	// Success 成功, 已调用管控端生成发布单，已发布的数量达到设置值，并且回执信息均为发布成功
	// Success, the control side has been called to generate the release order, the number of published has reached the set value, and the receipt information is published successfully
	Success = "Success"
	// Failed 失败，已调用管控端调用失败，后续不在生成新的ChangePod
	// Failed, the control side has been called to fail, and no new ChangePod will be generated later
	Failed = "Failed"
	// Suspend 暂停，已调用管控端生成发布单，触发预设的暂停策略，还需要继续发布，但是新的发布将会被暂停
	// Suspend, the control side has been called to generate the release order, triggering the preset pause policy, and still need to continue to release, but the new release will be suspended
	Suspend = "Suspend"
	// TimeOutPreThreshold 超时，已调用管控端生成发布单，但已发布的数量未达到设置值，超时使用未达到阈值的发布数量生成ChangePod
	// TimeOutPreThreshold Timeout, the control side has been called to generate the release order, but the number of published has not reached the set value, and the number of published that has not reached the threshold is used to generate ChangePod
	TimeOutPreThreshold = "TimeOutPreThreshold"
)

// ChangeWorkloadSpec defines the desired state of ChangeWorkload
type ChangeWorkloadSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	ChangeWorkloadId  string          `json:"changeWorkloadId"`
	ServiceName       string          `json:"serviceName"`
	Reversion         string          `json:"reversion"`
	Policies          []DefensePolicy `json:"policies,omitempty"`
	CountThreshold    int             `json:"countThreshold"`
	WaitTimeThreshold int             `json:"waitTimeThreshold"`
	CreateTime        string          `json:"createTime"`
	CreateTimeUnix    int64           `json:"createTimeUnix"`
	AppName           string          `json:"appName"`
}

type DefensePolicy struct {
	// TODO 暂停策略描述
}

type PodSummary struct {
	App       string `json:"app"`
	Hostname  string `json:"hostName"`
	Workspace string `json:"workSpace"`
	Pod       string `json:"pod"`
	Ip        string `json:"ip"`
	Namespace string `json:"namespace"`

	Verdict string `json:"verdict,omitempty"`
	Message string `json:"message,omitempty"`
}

// ChangeWorkloadStatus defines the observed state of ChangeWorkload
type ChangeWorkloadStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	DefensePreparingPods []PodSummary `json:"defensePreparingPods,omitempty"`
	DefenseCheckingPods  []PodSummary `json:"defenseCheckingPods,omitempty"`
	DefenseCheckPassPods []PodSummary `json:"defenseCheckPassPods,omitempty"`
	DefenseCheckFailPods []PodSummary `json:"defenseCheckFailPods,omitempty"`
	Status               string       `json:"status"`
	EntryTime            string       `json:"entryTime,omitempty"`
	EntryTimeUnix        int64        `json:"entryTimeUnix,omitempty"`
	UpdateTime           string       `json:"updateTime"`
	UpdateTimeUnix       int64        `json:"updateTimeUnix"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status",description="The status of the changeworkload"
//+kubebuilder:printcolumn:name="CreateTime",type="string",JSONPath=".spec.createTime",description="The create time of the changeworkload"

// ChangeWorkload is the Schema for the changeworkloads API
type ChangeWorkload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChangeWorkloadSpec   `json:"spec,omitempty"`
	Status ChangeWorkloadStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ChangeWorkloadList contains a list of ChangeWorkload
type ChangeWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ChangeWorkload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChangeWorkload{}, &ChangeWorkloadList{})
}
