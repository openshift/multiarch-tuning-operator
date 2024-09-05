package builder

import (
	ocpconfigv1 "github.com/openshift/api/config/v1"
)

// ImageDigestMirrorSetBuilder is a builder for ocpconfigv1.ImageDigestMirrorSet objects to be used only in unit tests.
type ImageDigestMirrorSetBuilder struct {
	imageDigestMirrorSet *ocpconfigv1.ImageDigestMirrorSet
}

// NewImageDigestMirrorSet returns a new ImageDigestMirrorSetBuilder to build ocpconfigv1.ImageDigestMirrorSet objects. It is meant to be used only in unit tests.
func NewImageDigestMirrorSet() *ImageDigestMirrorSetBuilder {
	return &ImageDigestMirrorSetBuilder{
		imageDigestMirrorSet: &ocpconfigv1.ImageDigestMirrorSet{},
	}
}

func (t *ImageDigestMirrorSetBuilder) WithImageDigestMirrors(values ...*ocpconfigv1.ImageDigestMirrors) *ImageDigestMirrorSetBuilder {
	for _, v := range values {
		if v == nil {
			panic("nil value passed to WithImageDigestMirrors")
		}
		t.imageDigestMirrorSet.Spec.ImageDigestMirrors = append(t.imageDigestMirrorSet.Spec.ImageDigestMirrors, *v)
	}
	return t
}

func (t *ImageDigestMirrorSetBuilder) WithName(name string) *ImageDigestMirrorSetBuilder {
	t.imageDigestMirrorSet.Name = name
	return t
}

func (t *ImageDigestMirrorSetBuilder) Build() *ocpconfigv1.ImageDigestMirrorSet {
	return t.imageDigestMirrorSet
}
