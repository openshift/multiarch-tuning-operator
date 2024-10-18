package builder

import v1 "k8s.io/api/core/v1"

// ContainerEnvBuilder is a builder for v1.EnvVar objects to be used only in unit tests.
type ContainerEnvBuilder struct {
	containerEnv v1.EnvVar
}

// NewContainerEnv returns a new ContainerEnvBuilder to build v1.EnvVar objects. It is meant to be used only in unit tests.
func NewContainerEnv() *ContainerEnvBuilder {
	return &ContainerEnvBuilder{
		containerEnv: v1.EnvVar{},
	}
}

func (e *ContainerEnvBuilder) WithName(name string) *ContainerEnvBuilder {
	e.containerEnv.Name = name
	return e
}

func (e *ContainerEnvBuilder) WithValue(value string) *ContainerEnvBuilder {
	e.containerEnv.Value = value
	return e
}

func (e *ContainerEnvBuilder) Build() v1.EnvVar {
	return e.containerEnv
}
