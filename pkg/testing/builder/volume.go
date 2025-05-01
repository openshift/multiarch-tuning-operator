package builder

import v1 "k8s.io/api/core/v1"

// VolumeBuilder is a builder for v1.Volume objects to be used only in unit tests.
type VolumeBuilder struct {
	volume *v1.Volume
}

// NewVolume returns a new VolumeBuilder to build v1.Volume objects. It is meant to be used only in unit tests.
func NewVolume() *VolumeBuilder {
	return &VolumeBuilder{
		volume: &v1.Volume{},
	}
}

func (v *VolumeBuilder) WithName(name string) *VolumeBuilder {
	v.volume.Name = name
	return v
}

func (v *VolumeBuilder) WithVolumeSourceHostPath(path string, pathType *v1.HostPathType) *VolumeBuilder {
	if v.volume.HostPath == nil {
		v.volume.HostPath = &v1.HostPathVolumeSource{}
	}
	v.volume.HostPath.Path = path
	v.volume.HostPath.Type = pathType
	return v
}

func (v *VolumeBuilder) WithVolumeSourceConfigmap(name string, values ...v1.KeyToPath) *VolumeBuilder {
	if v.volume.ConfigMap == nil {
		v.volume.ConfigMap = &v1.ConfigMapVolumeSource{}
	}
	v.volume.ConfigMap.LocalObjectReference = v1.LocalObjectReference{
		Name: name,
	}
	v.volume.ConfigMap.Items = values
	return v
}

func (v *VolumeBuilder) WithVolumeEmptyDir(value *v1.EmptyDirVolumeSource) *VolumeBuilder {
	if v.volume.EmptyDir == nil {
		v.volume.EmptyDir = &v1.EmptyDirVolumeSource{}
	}
	v.volume.EmptyDir = value
	return v
}

func (v *VolumeBuilder) WithVolumeProjectedSourcesSecretLocalObjectReference(names ...string) *VolumeBuilder {
	if v.volume.Projected == nil {
		v.volume.Projected = &v1.ProjectedVolumeSource{}
	}
	v.volume.Projected.Sources = make([]v1.VolumeProjection, len(names))
	for i, name := range names {
		v.volume.Projected.Sources[i] = v1.VolumeProjection{
			Secret: &v1.SecretProjection{
				LocalObjectReference: v1.LocalObjectReference{
					Name: name,
				},
			},
		}
	}
	return v
}

func (v *VolumeBuilder) WithVolumeProjectedDefaultMode(value *int32) *VolumeBuilder {
	if v.volume.Projected == nil {
		v.volume.Projected = &v1.ProjectedVolumeSource{}
	}
	v.volume.Projected.DefaultMode = value
	return v
}

func (v *VolumeBuilder) Build() *v1.Volume {
	return v.volume
}
