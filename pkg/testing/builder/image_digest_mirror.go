package builder

import (
	ocpconfigv1 "github.com/openshift/api/config/v1"
)

// ImageDigestMirrorsBuilder is a builder for ocpconfigv1.imageDigestMirrorset objects to be used only in unit tests.
type ImageDigestMirrorsBuilder struct {
	imageDigestMirrors *ocpconfigv1.ImageDigestMirrors
}

// NewImageDigestMirrors returns a new ImageDigestMirrorsBuilder to build ocpconfigv1.imageDigestMirrorset objects. It is meant to be used only in unit tests.
func NewImageDigestMirrors() *ImageDigestMirrorsBuilder {
	return &ImageDigestMirrorsBuilder{
		imageDigestMirrors: &ocpconfigv1.ImageDigestMirrors{},
	}
}

func (t *ImageDigestMirrorsBuilder) WithMirrors(values ...ocpconfigv1.ImageMirror) *ImageDigestMirrorsBuilder {
	t.imageDigestMirrors.Mirrors = append(t.imageDigestMirrors.Mirrors, values...)
	return t
}

func (t *ImageDigestMirrorsBuilder) WithSource(registry string) *ImageDigestMirrorsBuilder {
	t.imageDigestMirrors.Source = registry
	return t
}

func (t *ImageDigestMirrorsBuilder) WithMirrorSourcePolicy(policy ocpconfigv1.MirrorSourcePolicy) *ImageDigestMirrorsBuilder {
	t.imageDigestMirrors.MirrorSourcePolicy = policy
	return t
}

func (t *ImageDigestMirrorsBuilder) WithMirrorAllowContactingSource() *ImageDigestMirrorsBuilder {
	return t.WithMirrorSourcePolicy(ocpconfigv1.AllowContactingSource)
}

func (t *ImageDigestMirrorsBuilder) WithMirrorNeverContactSource() *ImageDigestMirrorsBuilder {
	return t.WithMirrorSourcePolicy(ocpconfigv1.NeverContactSource)
}

func (t *ImageDigestMirrorsBuilder) Build() *ocpconfigv1.ImageDigestMirrors {
	return t.imageDigestMirrors
}
