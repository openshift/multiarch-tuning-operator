package builder

import (
	"crypto/sha256"
	"encoding/hex"

	v1 "k8s.io/api/core/v1"
)

// ContainerBuilder is a builder for v1.Container objects to be used only in unit tests.
type ContainerBuilder struct {
	container *v1.Container
}

// NewContainer returns a new ContainerBuilder to build v1.Container objects. It is meant to be used only in unit tests.
func NewContainer() *ContainerBuilder {
	return &ContainerBuilder{
		container: &v1.Container{},
	}
}

func (c *ContainerBuilder) WithImage(image string) *ContainerBuilder {
	hasher := sha256.New()
	hasher.Write([]byte(image))

	c.container = &v1.Container{
		Image: image,
		Name:  hex.EncodeToString(hasher.Sum(nil))[:63], // hash of the image name (63 is max)
	}
	return c
}

func (c *ContainerBuilder) WithSecurityContext(securityContext *v1.SecurityContext) *ContainerBuilder {
	c.container.SecurityContext = securityContext
	return c
}

func (c *ContainerBuilder) WithVolumeMounts(volumeMounts ...*v1.VolumeMount) *ContainerBuilder {
	for i := range volumeMounts {
		if volumeMounts[i] == nil {
			panic("nil value passed to WithVolumeMounts")
		}
		c.container.VolumeMounts = append(c.container.VolumeMounts, *volumeMounts[i])
	}
	return c
}

func (c *ContainerBuilder) WithEnv(values ...*v1.EnvVar) *ContainerBuilder {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithEnv")
		}
		c.container.Env = append(c.container.Env, *values[i])
	}

	return c
}

func (c *ContainerBuilder) WithPorts(values ...v1.ContainerPort) *ContainerBuilder {
	c.container.Ports = append(c.container.Ports, values...)
	return c
}

func (c *ContainerBuilder) WithPortsContainerPort(ports ...int32) *ContainerBuilder {
	c.container.Ports = make([]v1.ContainerPort, len(ports))
	for i, port := range ports {
		c.container.Ports[i] = v1.ContainerPort{
			ContainerPort: port,
		}
	}
	return c
}

func (c *ContainerBuilder) Build() *v1.Container {
	return c.container
}
