package builder

import v1 "k8s.io/api/core/v1"

// VolumeBuilder is a builder for v1.Volume objects to be used only in unit tests.
type VolumeBuilder struct {
	volume v1.Volume
}

// NewVolume returns a new VolumeBuilder to build v1.Volume objects. It is meant to be used only in unit tests.
func NewVolume() *VolumeBuilder {
	return &VolumeBuilder{
		volume: v1.Volume{},
	}
}

func (v *VolumeBuilder) WithName(name string) *VolumeBuilder {
	v.volume.Name = name
	return v
}

func (v *VolumeBuilder) WithVolumeSourceHostPath(path string, pathType *v1.HostPathType) *VolumeBuilder {
	if v.volume.VolumeSource.HostPath == nil {
		v.volume.VolumeSource.HostPath = &v1.HostPathVolumeSource{}
	}
	v.volume.VolumeSource.HostPath.Path = path
	v.volume.VolumeSource.HostPath.Type = pathType
	return v
}

func (v *VolumeBuilder) Build() v1.Volume {
	return v.volume
}
