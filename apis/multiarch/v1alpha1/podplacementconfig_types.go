/*
Copyright 2023 Red Hat, Inc.

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

// +kubebuilder:validation:Enum=Normal;Debug;Trace;TraceAll
type LogVerbosityLevel string

const (
	LogVerbosityLevelNormal   LogVerbosityLevel = "Normal"
	LogVerbosityLevelDebug    LogVerbosityLevel = "Debug"
	LogVerbosityLevelTrace    LogVerbosityLevel = "Trace"
	LogVerbosityLevelTraceAll LogVerbosityLevel = "TraceAll"
)

// PodPlacementConfigSpec defines the desired state of PodPlacementConfig
type PodPlacementConfigSpec struct {
	// LogVerbosity is the log level for the pod placement controller
	// Valid values are: "Normal", "Debug", "Trace", "TraceAll".
	// Defaults to "Normal".
	// +optional
	// +kubebuilder:default=Normal
	LogVerbosity LogVerbosityLevel `json:"logVerbosity,omitempty"`

	// NamespaceSelector decides whether to run the admission control policy on an object based
	// on whether the namespace for that object matches the selector. If the
	// object itself is a namespace, the matching is performed on
	// object.metadata.labels. If the object is another cluster scoped resource,
	// it never skips the policy.
	//
	// For example, to run the webhook on any objects whose namespace is not
	// associated with "runlevel" of "0" or "1";  you will set the selector as
	// follows:
	// "namespaceSelector": {
	//   "matchExpressions": [
	//     {
	//       "key": "runlevel",
	//       "operator": "NotIn",
	//       "values": [
	//         "0",
	//         "1"
	//       ]
	//     }
	//   ]
	// }
	//
	// If instead you want to only run the policy on any objects whose
	// namespace is associated with the "environment" of "prod" or "staging";
	// you will set the selector as follows:
	// "namespaceSelector": {
	//   "matchExpressions": [
	//     {
	//       "key": "environment",
	//       "operator": "In",
	//       "values": [
	//         "prod",
	//         "staging"
	//       ]
	//     }
	//   ]
	// }
	//
	// See
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
	// for more examples of label selectors.
	//
	// Default to the empty LabelSelector, which matches everything.
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
}

// PodPlacementConfigStatus defines the observed state of PodPlacementConfig
type PodPlacementConfigStatus struct {
	// Conditions represents the latest available observations of a PodPlacementConfig's current state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=podplacementconfigs,scope=Cluster
// PodPlacementConfig is the Schema for the podplacementconfigs API
type PodPlacementConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodPlacementConfigSpec   `json:"spec,omitempty"`
	Status PodPlacementConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PodPlacementConfigList contains a list of PodPlacementConfig
type PodPlacementConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodPlacementConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PodPlacementConfig{}, &PodPlacementConfigList{})
}
