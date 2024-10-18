package builder

import v1 "k8s.io/api/core/v1"

// ServiceBuilder is a builder for v1.Service objects to be used only in unit tests.
type ServiceBuilder struct {
	service *v1.Service
}

// Newservice returns a new ServiceBuilder to build v1.Service objects. It is meant to be used only in unit tests.
func NewService() *ServiceBuilder {
	return &ServiceBuilder{
		service: &v1.Service{},
	}
}

func (s *ServiceBuilder) WithName(name string) *ServiceBuilder {
	s.service.Name = name
	return s
}

func (s *ServiceBuilder) WithNamespace(namespace string) *ServiceBuilder {
	s.service.Namespace = namespace
	return s
}

func (s *ServiceBuilder) WithPorts(values ...v1.ServicePort) *ServiceBuilder {
	s.service.Spec.Ports = append(s.service.Spec.Ports, values...)
	return s
}

func (s *ServiceBuilder) WithSelector(entries map[string]string) *ServiceBuilder {
	if s.service.Spec.Selector == nil && len(entries) > 0 {
		s.service.Spec.Selector = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		s.service.Spec.Selector[k] = v
	}
	return s
}

func (s *ServiceBuilder) Build() *v1.Service {
	return s.service
}
