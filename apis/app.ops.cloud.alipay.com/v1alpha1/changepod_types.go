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
	// ExecuteInit 初始化状态，未开始执行
	// ExecuteInit init status, not start execute
	ExecuteInit = ""
	// PreWait 预执行等待提交状态
	// PreWait pre execute wait submit status
	PreWait = "PreWait"
	// PreSubmitted 预执行已提交状态
	// PreSubmitted pre execute submitted status
	PreSubmitted = "PreSubmitted"
	// PreTimeout 预执行超时状态
	// PreTimeout pre execute timeout status
	PreTimeout = "PreTimeout"
	// PreFailed 预执行失败状态
	// PreFailed pre execute failed status
	PreFailed = "PreFailed"
	// PostWait 后执行等待提交状态
	// PostWait post execute wait submit status
	PostWait = "PostWait"
	// PostSubmitted 后执行已提交状态
	// PostSubmitted post execute submitted status
	PostSubmitted = "PostSubmitted"
	// PostFinish 后执行完成状态
	// PostFinish post execute finish status
	PostFinish = "PostFinish"
	// PostTimeout 后执行超时状态
	// PostTimeout post execute timeout status
	PostTimeout = "PostTimeout"
	// PostFailed 后执行失败状态
	// PostFailed post execute failed status
	PostFailed = "PostFailed"
	// ExecuteDone 执行完成状态
	// ExecuteDone execute done status
	ExecuteDone = "ExecuteDone"
)

// ChangePodSpec defines the desired state of ChangePod
type ChangePodSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	PodInfos         []PodSummary `json:"podInfos"`
	CreateTime       string       `json:"createTime"`
	CreateTimeUnix   int64        `json:"createTimeUnix"`
	ChangeWorkloadId string       `json:"changeWorkloadId"`
}

// ChangePodStatus defines the observed state of ChangePod
type ChangePodStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status         string       `json:"status"`
	UpdateTime     string       `json:"updateTime"`
	UpdateTimeUnix int64        `json:"updateTimeUnix"`
	PodResults     []PodSummary `json:"podResults,omitempty"`

	Message     string `json:"message,omitempty"`
	ChangePodId string `json:"changePodId,omitempty"`

	PreSubmitTime        string `json:"preSubmitTime,omitempty"`
	PreSubmitTimeUnix    int64  `json:"preSubmitTimeUnix,omitempty"`
	PostSubmitTime       string `json:"postSubmitTime,omitempty"`
	PostSubmitTimeUnix   int64  `json:"postSubmitTimeUnix,omitempty"`
	PreTimeoutThreshold  int    `json:"preTimeoutThreshold,omitempty"`
	PostTimeoutThreshold int    `json:"postTimeOutThreshold,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status",description="The status of the changepod"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.message",description="The message of the changepod"
//+kubebuilder:printcolumn:name="CreateTime",type="string",JSONPath=".spec.createTime",description="The create time of the changepod"

// ChangePod is the Schema for the changepods API
type ChangePod struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChangePodSpec   `json:"spec,omitempty"`
	Status ChangePodStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ChangePodList contains a list of ChangePod
type ChangePodList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ChangePod `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChangePod{}, &ChangePodList{})
}
