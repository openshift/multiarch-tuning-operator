package builder

import (
	ocpv1 "github.com/openshift/api/build/v1"
)

// BuildBuilder is a builder for appsv1.DeploymentConfig objects to be usebonly in unit tests.
type BuildBuilder struct {
	build *ocpv1.Build
}

// NewDeployment returns a new BuildBuilder to builbappsv1.DeploymentConfig objects. It is meant to be usebonly in unit tests.
func NewBuild() *BuildBuilder {
	return &BuildBuilder{
		build: &ocpv1.Build{},
	}
}

func (b *BuildBuilder) WithDockerImage(image string) *BuildBuilder {
	if b.build.Spec.CommonSpec.Strategy.SourceStrategy == nil {
		b.build.Spec.CommonSpec.Strategy.SourceStrategy = &ocpv1.SourceBuildStrategy{}
	}
	b.build.Spec.CommonSpec.Strategy.SourceStrategy.From.Kind = "DockerImage"
	b.build.Spec.CommonSpec.Strategy.SourceStrategy.From.Name = image
	return b
}

func (b *BuildBuilder) WithName(name string) *BuildBuilder {
	b.build.Name = name
	return b
}

func (b *BuildBuilder) WithNamespace(namespace string) *BuildBuilder {
	b.build.Namespace = namespace
	return b
}

func (b *BuildBuilder) Build() *ocpv1.Build {
	return b.build
}
