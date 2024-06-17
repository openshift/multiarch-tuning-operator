package builder

import v1 "k8s.io/api/core/v1"

// ServicePortBuilder is a builder for v1.ServicePort objects to be used only in unit tests.
type ServicePortBuilder struct {
	servicePort v1.ServicePort
}

// NewServicePort returns a new ServicePortBuilder to build v1.ServicePort objects. It is meant to be used only in unit tests.
func NewServicePort() *ServicePortBuilder {
	return &ServicePortBuilder{
		servicePort: v1.ServicePort{},
	}
}

func (s *ServicePortBuilder) WithProtocol(value v1.Protocol) *ServicePortBuilder {
	s.servicePort.Protocol = value
	return s
}

func (s *ServicePortBuilder) WithTCPProtocol() *ServicePortBuilder {
	return s.WithProtocol("TCP")
}

func (s *ServicePortBuilder) WithPort(value int32) *ServicePortBuilder {
	s.servicePort.Port = value

	return s
}

func (s *ServicePortBuilder) WithTargetPort(value int32) *ServicePortBuilder {
	s.servicePort.TargetPort.IntVal = value
	return s
}

func (s *ServicePortBuilder) Build() v1.ServicePort {
	return s.servicePort
}
