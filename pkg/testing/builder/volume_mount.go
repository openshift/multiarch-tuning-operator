package builder

import v1 "k8s.io/api/core/v1"

// VolumeMountBuilder is a builder for v1.VolumeMount objects to be used only in unit tests.
type VolumeMountBuilder struct {
	volumeMount *v1.VolumeMount
}

// NewVolumeMount returns a new VolumeMountBuilder to build v1.VolumeMount objects. It is meant to be used only in unit tests.
func NewVolumeMount() *VolumeMountBuilder {
	return &VolumeMountBuilder{
		volumeMount: &v1.VolumeMount{},
	}
}

func (v *VolumeMountBuilder) WithName(name string) *VolumeMountBuilder {
	v.volumeMount.Name = name
	return v
}

func (v *VolumeMountBuilder) WithMountPath(path string) *VolumeMountBuilder {
	v.volumeMount.MountPath = path
	return v
}

func (v *VolumeMountBuilder) WithReadOnly() *VolumeMountBuilder {
	v.volumeMount.ReadOnly = true
	return v
}

func (v *VolumeMountBuilder) Build() *v1.VolumeMount {
	return v.volumeMount
}
