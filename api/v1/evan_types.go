/*
Copyright 2024.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Evan is the Schema for the evans API
// +kubebuilder:printcolumn:name="AvailableReplicas",type="integer",JSONPath=".status.availableReplicas"
type Evan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EvanSpec   `json:"spec,omitempty"`
	Status EvanStatus `json:"status,omitempty"`
}

type DeploymentConfig struct {
	// +optional
	Name string `json:"name,omitempty"`
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// +optional
	Image string `json:"image"`
}

type ServiceConfig struct {
	// +optional
	Name string `json:"name,omitempty"`
	// +optional
	Type corev1.ServiceType `json:"type,omitempty"`
	Port int32              `json:"port,omitempty"`
	// +optional
	TargetPort int32 `json:"targetPort,omitempty"`
	// +optional
	NodePort int32 `json:"nodePort,omitempty"`
}

type DeletionPolicy string

const (
	DeletionPolicyDelete  DeletionPolicy = "Delete"
	DeletionPolicyWipeOut DeletionPolicy = "WipeOut"
)

// EvanSpec defines the desired state of Evan
type EvanSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	DeploymentConfig DeploymentConfig `json:"deploymentConfig,omitempty"`
	ServiceConfig    ServiceConfig    `json:"serviceConfig,omitempty"`
	DeletionPolicy   DeletionPolicy   `json:"deletionPolicy,omitempty"`
}

// EvanStatus defines the observed state of Evan
type EvanStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	AvailableReplicas int32 `json:"availableReplicas"`
}

//+kubebuilder:object:root=true

// EvanList contains a list of Evan
type EvanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Evan `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Evan{}, &EvanList{})
}
