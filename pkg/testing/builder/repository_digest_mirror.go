package builder

import (
	ocpoperatorv1alpha1 "github.com/openshift/api/operator/v1alpha1"
)

// RepositoryDigestMirrorsBuilder is a builder for ocpoperatorv1alpha1.RepositoryDigestMirrorset objects to be used only in unit tests.
type RepositoryDigestMirrorsBuilder struct {
	repositoryDigestMirrors *ocpoperatorv1alpha1.RepositoryDigestMirrors
}

// NewRepositoryDigestMirrors returns a new RepositoryDigestMirrorsBuilder to build ocpoperatorv1alpha1.RepositoryDigestMirrorset objects. It is meant to be used only in unit tests.
func NewRepositoryDigestMirrors() *RepositoryDigestMirrorsBuilder {
	return &RepositoryDigestMirrorsBuilder{
		repositoryDigestMirrors: &ocpoperatorv1alpha1.RepositoryDigestMirrors{},
	}
}

func (t *RepositoryDigestMirrorsBuilder) WithMirrors(values ...string) *RepositoryDigestMirrorsBuilder {
	t.repositoryDigestMirrors.Mirrors = append(t.repositoryDigestMirrors.Mirrors, values...)
	return t
}

func (t *RepositoryDigestMirrorsBuilder) WithSource(registry string) *RepositoryDigestMirrorsBuilder {
	t.repositoryDigestMirrors.Source = registry
	return t
}

func (t *RepositoryDigestMirrorsBuilder) Build() *ocpoperatorv1alpha1.RepositoryDigestMirrors {
	return t.repositoryDigestMirrors
}
