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
	"k8s.io/apimachinery/pkg/util/intstr"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type TargetType string

const (
	TargetTypeWorkload TargetType = "workload"
)

// ChangeDefenseSpec defines the desired state of ChangeDefense
type ChangeDefenseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Target indicates the resource that needs change defense
	Target *TargetRef `json:"target"`
	// DefenseStrategy indicates the defense strategy on targeted resource changes
	DefenseStrategy *DefenseStrategy `json:"defenseStrategy"`
	// RiskPolicy indicates what to do on risks discovered by defense strategy
	// +kubebuilder:validation:Optional
	RiskPolicy *RiskPolicy `json:"riskPolicy"`
}

// TargetRef defines the object that needs change defense
type TargetRef struct {
	// Type of change resource target
	Type TargetType `json:"type"`
	// ObjectRef references the targeted kubernetes object
	ObjectRef *ObjectRef `json:"objectRef"`
}

// DefenseStrategy defines the defense strategy on resource changes
type DefenseStrategy struct {
	// Workload indicates the defense strategy on targeted workload changes
	// +kubebuilder:validation:Optional
	Workload WorkloadDefenseStrategy `json:"workload"`
}

// Workload defines the defense strategy on workload changes
type WorkloadDefenseStrategy struct {
	// Steps define the order of phases to execute defense in batches(e.g. 20%, 40%, 60%, 80%, 100%)
	Steps []WorkloadDefenseStep `json:"steps"`
}

type WorkloadDefenseStep struct {
	Partition intstr.IntOrString `json:"partition"`
	// +kubebuilder:validation:Optional
	CheckAfterComplete *int32 `json:"checkAfterComplete"`
}

// RiskPolicy defines what to do on risks discovered by defense strategy
type RiskPolicy struct {
}

type ObjectRef struct {
	// API Version of the referent
	APIVersion string `json:"apiVersion"`
	// Kind of the referent
	Kind string `json:"kind"`
	// Name of the referent
	Name string `json:"name"`
}

type DefensePhase string
const (
	DefensePhaseInitial     DefensePhase = "Initial"
	DefensePhasePreCheck    DefensePhase = "PreCheck"
	DefensePhaseProgressing DefensePhase = "Progressing"
	DefensePhaseObserving 	DefensePhase = "Observing"
	DefensePhasePostCheck   DefensePhase = "PostCheck"
	DefensePhasePassed      DefensePhase = "Passed"
	DefensePhaseFailed      DefensePhase = "Failed"
	DefensePhaseSkipped     DefensePhase = "Skipped"
)

// ChangeDefenseStatus defines the observed state of ChangeDefense
type ChangeDefenseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +kubebuilder:validation:Optional
	CurrentExecutionID string `json:"currentExecutionID"`
	// +kubebuilder:validation:Optional
	Verdict string `json:"verdict"`
	// +kubebuilder:validation:Optional
	Phase DefensePhase `json:"phase"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ChangeDefense is the Schema for the changedefenses API
type ChangeDefense struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChangeDefenseSpec   `json:"spec,omitempty"`
	Status ChangeDefenseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ChangeDefenseList contains a list of ChangeDefense
type ChangeDefenseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ChangeDefense `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChangeDefense{}, &ChangeDefenseList{})
}
