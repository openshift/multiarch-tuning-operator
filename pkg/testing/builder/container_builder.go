package builder

import (
	"crypto/sha256"
	"encoding/hex"

	v1 "k8s.io/api/core/v1"
)

// ContainerBuilder is a builder for v1.Container objects to be used only in unit tests.
type ContainerBuilder struct {
	container v1.Container
}

// NewContainer returns a new ContainerBuilder to build v1.Container objects. It is meant to be used only in unit tests.
func NewContainer() *ContainerBuilder {
	return &ContainerBuilder{
		container: v1.Container{},
	}
}

func (c *ContainerBuilder) WithImage(image string) *ContainerBuilder {
	hasher := sha256.New()
	hasher.Write([]byte(image))

	c.container = v1.Container{
		Image: image,
		Name:  hex.EncodeToString(hasher.Sum(nil))[:63], // hash of the image name (63 is max)
	}
	return c
}

func (c *ContainerBuilder) WithSecurityContext(securityContext *v1.SecurityContext) *ContainerBuilder {
	c.container.SecurityContext = securityContext
	return c
}

func (c *ContainerBuilder) WithVolumeMounts(volumeMounts ...v1.VolumeMount) *ContainerBuilder {
	c.container.VolumeMounts = volumeMounts
	return c
}

func (c *ContainerBuilder) Build() v1.Container {
	return c.container
}
