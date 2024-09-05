package builder

import (
	ocpoperatorv1alpha1 "github.com/openshift/api/operator/v1alpha1"
)

// ImageContentSourcePolicyBuilder is a builder for ocpoperatorv1alpha1.ImageContentSourcePolicy objects to be used only in unit tests.
type ImageContentSourcePolicyBuilder struct {
	imageContentSourcePolicy *ocpoperatorv1alpha1.ImageContentSourcePolicy
}

// NewImageContentSourcePolicy returns a new ImageContentSourcePolicyBuilder to build ocpoperatorv1alpha1.ImageContentSourcePolicy objects. It is meant to be used only in unit tests.
func NewImageContentSourcePolicy() *ImageContentSourcePolicyBuilder {
	return &ImageContentSourcePolicyBuilder{
		imageContentSourcePolicy: &ocpoperatorv1alpha1.ImageContentSourcePolicy{},
	}
}

func (t *ImageContentSourcePolicyBuilder) WithRepositoryDigestMirrors(values ...*ocpoperatorv1alpha1.RepositoryDigestMirrors) *ImageContentSourcePolicyBuilder {
	for _, v := range values {
		if v == nil {
			panic("nil value passed to WithRepositoryDigestMirrors")
		}
		t.imageContentSourcePolicy.Spec.RepositoryDigestMirrors = append(t.imageContentSourcePolicy.Spec.RepositoryDigestMirrors, *v)
	}
	return t
}

func (t *ImageContentSourcePolicyBuilder) WithName(name string) *ImageContentSourcePolicyBuilder {
	t.imageContentSourcePolicy.Name = name
	return t
}

func (t *ImageContentSourcePolicyBuilder) Build() *ocpoperatorv1alpha1.ImageContentSourcePolicy {
	return t.imageContentSourcePolicy
}
