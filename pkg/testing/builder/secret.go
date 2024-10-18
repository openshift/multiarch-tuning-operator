package builder

import v1 "k8s.io/api/core/v1"

// SecretBuilder is a builder for v1.Secret objects to be used only in unit tests.
type SecretBuilder struct {
	secret *v1.Secret
}

// NewSecret returns a new SecretBuilder to build v1.Secret objects. It is meant to be used only in unit tests.
func NewSecret() *SecretBuilder {
	return &SecretBuilder{
		secret: &v1.Secret{},
	}
}

func (s *SecretBuilder) WithName(name string) *SecretBuilder {
	s.secret.Name = name
	return s
}

func (s *SecretBuilder) WithNameSpace(namespace string) *SecretBuilder {
	s.secret.Namespace = namespace
	return s
}

func (s *SecretBuilder) WithData(entries map[string][]byte) *SecretBuilder {
	if s.secret.Data == nil && len(entries) > 0 {
		s.secret.Data = make(map[string][]byte, len(entries))
	}
	for k, v := range entries {
		s.secret.Data[k] = v
	}
	return s
}

func (s *SecretBuilder) WithType(value v1.SecretType) *SecretBuilder {
	s.secret.Type = value
	return s
}

func (s *SecretBuilder) WithDockerConfigJsonType() *SecretBuilder {
	return s.WithType("kubernetes.io/dockerconfigjson")
}

func (s *SecretBuilder) WithOpaqueType() *SecretBuilder {
	return s.WithType("Opaque")
}

func (s *SecretBuilder) Build() *v1.Secret {
	return s.secret
}
