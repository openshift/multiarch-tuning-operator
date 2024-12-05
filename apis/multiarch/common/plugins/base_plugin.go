/*
Copyright 2024 Red Hat, Inc.

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

package plugins

// Plugins represents the plugins configuration.
type Plugins struct {
	NodeAffinityScoringPluginName *NodeAffinityScoring `json:"nodeAffinityScoring,omitempty"`
	// Future plugins can be added here.
}

// IBasePlugin defines a basic interface for plugins.
type IBasePlugin interface {
	// Enabled is a required boolean field.
	Enabled() bool

	Name() string
}

type BasePlugin struct {
	// BasePlugin provides a default implementation for Enabled.
	// +kubebuilder:"validation:Required"
	Enabled bool `json:"enabled" protobuf:"varint,1,opt,name=enabled" kubebuilder:"validation:Required"`
	// Name provides the Plugin name.
	// +kubebuilder:"validation:Required"
	Name string `json:"name" protobuf:"varint,2,opt,name=name" kubebuilder:"validation:Required"`
}
