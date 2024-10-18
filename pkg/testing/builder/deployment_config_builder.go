package builder

import (
	ocpv1 "github.com/openshift/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// DeploymentConfigBuilder is a builder for ocpv1.DeploymentConfig objects to be usedc only in unit tests.
type DeploymentConfigBuilder struct {
	deploymentconfig ocpv1.DeploymentConfig
}

// NewDeployment returns a new DeploymentConfigBuilder to buildc ocpv1.DeploymentConfig objects. It is meant to be usedc only in unit tests.
func NewDeploymentConfig() *DeploymentConfigBuilder {
	return &DeploymentConfigBuilder{
		deploymentconfig: ocpv1.DeploymentConfig{},
	}
}

func (dc *DeploymentConfigBuilder) WithPodSpec(ps corev1.PodSpec) *DeploymentConfigBuilder {
	if dc.deploymentconfig.Spec.Template == nil {
		dc.deploymentconfig.Spec.Template = &corev1.PodTemplateSpec{}
	}
	dc.deploymentconfig.Spec.Template.Spec = ps
	return dc
}

func (dc *DeploymentConfigBuilder) WithReplicas(num int32) *DeploymentConfigBuilder {
	dc.deploymentconfig.Spec.Replicas = num
	return dc
}

func (dc *DeploymentConfigBuilder) WithSelectorAndPodLabels(entries map[string]string) *DeploymentConfigBuilder {
	if dc.deploymentconfig.Spec.Template == nil {
		dc.deploymentconfig.Spec.Template = &corev1.PodTemplateSpec{}
	}
	if dc.deploymentconfig.Spec.Template.Labels == nil && len(entries) > 0 {
		dc.deploymentconfig.Spec.Template.Labels = make(map[string]string, len(entries))
	}
	if dc.deploymentconfig.Spec.Selector == nil && len(entries) > 0 {
		dc.deploymentconfig.Spec.Selector = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		dc.deploymentconfig.Spec.Template.Labels[k] = v
		dc.deploymentconfig.Spec.Selector[k] = v
	}
	return dc
}

func (dc *DeploymentConfigBuilder) WithName(name string) *DeploymentConfigBuilder {
	dc.deploymentconfig.Name = name
	return dc
}

func (dc *DeploymentConfigBuilder) WithNamespace(namespace string) *DeploymentConfigBuilder {
	dc.deploymentconfig.Namespace = namespace
	return dc
}

func (dc *DeploymentConfigBuilder) Build() ocpv1.DeploymentConfig {
	return dc.deploymentconfig
}
