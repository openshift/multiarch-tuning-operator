package builder

import (
	v1 "k8s.io/api/core/v1"
)

// SecurityContextBuilder is a builder for v1.SecurityContext objects to be used only in unit tests.
type SecurityContextBuilder struct {
	securityContext v1.SecurityContext
}

// NewSecurityContext returns a new SecurityContextBuilder to build v1.SecurityContext objects. It is meant to be used only in unit tests.
func NewSecurityContext() *SecurityContextBuilder {
	return &SecurityContextBuilder{
		securityContext: v1.SecurityContext{},
	}
}

func (s *SecurityContextBuilder) WithPrivileged(privileged *bool) *SecurityContextBuilder {
	s.securityContext.Privileged = privileged
	return s
}

func (s *SecurityContextBuilder) WithRunAsGroup(group *int64) *SecurityContextBuilder {
	s.securityContext.RunAsGroup = group
	return s
}

func (s *SecurityContextBuilder) WithRunAsUSer(user *int64) *SecurityContextBuilder {
	s.securityContext.RunAsUser = user
	return s
}

func (s *SecurityContextBuilder) WithSeccompProfileType(seccompProfileType v1.SeccompProfileType) *SecurityContextBuilder {
	s.securityContext.SeccompProfile.Type = seccompProfileType
	return s
}

func (s *SecurityContextBuilder) Build() v1.SecurityContext {
	return s.securityContext
}
