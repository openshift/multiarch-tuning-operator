package builder

import (
	v1 "k8s.io/api/rbac/v1"
)

// DeploymentBuilder is a builder for v1.Subject objects to be used only in unit tests.
type SubjectBuilder struct {
	subject v1.Subject
}

// NewSubject returns a new DeploymentBuilder to build v1.Subject objects. It is meant to be used only in unit tests.
func NewSubject() *SubjectBuilder {
	return &SubjectBuilder{
		subject: v1.Subject{},
	}
}

func (s *SubjectBuilder) WithKind(kind string) *SubjectBuilder {
	s.subject.Kind = kind
	return s
}

func (s *SubjectBuilder) WithName(name string) *SubjectBuilder {
	s.subject.Name = name
	return s
}

func (s *SubjectBuilder) WithNamespace(namespace string) *SubjectBuilder {
	s.subject.Namespace = namespace
	return s
}

func (s *SubjectBuilder) Build() v1.Subject {
	return s.subject
}
