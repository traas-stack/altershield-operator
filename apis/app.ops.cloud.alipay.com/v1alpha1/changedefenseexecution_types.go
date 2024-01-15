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

// ChangeDefenseExecutionSpec defines the desired state of ChangeDefenseExecution
type ChangeDefenseExecutionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ID is the defense execution unique identifier in altershield
	ID string `json:"id,omitempty"`
	// Target indicates the resource that needs change defense
	Target *TargetRef `json:"target"`
	// DefenseStrategy indicates the defense strategy on targeted resource changes
	DefenseStrategy *DefenseStrategy `json:"defenseStrategy"`
	// RiskPolicy indicates what to do on risks discovered by defense strategy
	// +kubebuilder:validation:Optional
	RiskPolicy *RiskPolicy `json:"riskPolicy"`
}

type PodInfo struct {
	Name string `json:"name"`
	IP string `json:"ip"`
	UID string `json:"uid"`
}

type DefenseTargetStatus struct {
	// PodsBatches records the pods checked at each batch
	// +kubebuilder:validation:Optional
	PodBatches []BatchPodInfo `json:"podBatches"`
}

type BatchPodInfo struct {
	// Pods records the pods checked in single batch
	// +kubebuilder:validation:Optional
	Pods []PodInfo `json:"pods"`
}

type DefenseStatus struct {
	// CurrentBatch is the latest batch number that change defense is executing on, starts from 1
	CurrentBatch int `json:"currentBatch"`
	// Phase describes the life-cycle of defense execution in this batch
	Phase DefensePhase `json:"phase"`
	Verdict string `json:"verdict"`
	// TargetStatus is observed status of change defense target
	// +kubebuilder:validation:Optional
	TargetStatus DefenseTargetStatus `json:"targetStatus"`
	// +kubebuilder:validation:Optional
	NodeID string `json:"nodeID"`
	LastTransitionTime   *metav1.Time    `json:"lastTransitionTime,omitempty"`
}

// ChangeDefenseExecutionStatus defines the observed state of ChangeDefenseExecution
type ChangeDefenseExecutionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// DefenseStatus records the status information of execution
	DefenseStatus DefenseStatus `json:"defenseStatus"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ChangeDefenseExecution is the Schema for the changedefenseexecutions API
type ChangeDefenseExecution struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChangeDefenseExecutionSpec   `json:"spec,omitempty"`
	Status ChangeDefenseExecutionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ChangeDefenseExecutionList contains a list of ChangeDefenseExecution
type ChangeDefenseExecutionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ChangeDefenseExecution `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChangeDefenseExecution{}, &ChangeDefenseExecutionList{})
}
