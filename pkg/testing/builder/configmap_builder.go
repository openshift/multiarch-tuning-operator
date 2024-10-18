package builder

import v1 "k8s.io/api/core/v1"

// ConfigMapBuilder is a builder for v1.ConfigMap objects to be used only in unit tests.
type ConfigMapBuilder struct {
	configmap v1.ConfigMap
}

// NewConfigMap returns a new ConfigMapBuilder to build v1.ConfigMap objects. It is meant to be used only in unit tests.
func NewConfigMap() *ConfigMapBuilder {
	return &ConfigMapBuilder{
		configmap: v1.ConfigMap{},
	}
}

func (s *ConfigMapBuilder) WithName(name string) *ConfigMapBuilder {
	s.configmap.Name = name
	return s
}

func (s *ConfigMapBuilder) WithNamespace(namespace string) *ConfigMapBuilder {
	s.configmap.Namespace = namespace
	return s
}

func (s *ConfigMapBuilder) WithData(entries map[string]string) *ConfigMapBuilder {
	if s.configmap.Data == nil && len(entries) > 0 {
		s.configmap.Data = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		s.configmap.Data[k] = v
	}
	return s
}

func (s *ConfigMapBuilder) WithBinaryData(entries map[string][]byte) *ConfigMapBuilder {
	if s.configmap.BinaryData == nil && len(entries) > 0 {
		s.configmap.BinaryData = make(map[string][]byte, len(entries))
	}
	for k, v := range entries {
		s.configmap.BinaryData[k] = v
	}
	return s
}

func (s *ConfigMapBuilder) WithLabels(entries map[string]string) *ConfigMapBuilder {
	if s.configmap.Labels == nil && len(entries) > 0 {
		s.configmap.Labels = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		s.configmap.Labels[k] = v
	}

	return s
}

func (s *ConfigMapBuilder) Build() v1.ConfigMap {
	return s.configmap
}
