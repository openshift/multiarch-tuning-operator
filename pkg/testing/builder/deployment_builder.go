package builder

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeploymentBuilder is a builder for appsv1.Deployment objects to be used only in unit tests.
type DeploymentBuilder struct {
	deployment appsv1.Deployment
}

// NewDeployment returns a new DeploymentBuilder to build appsv1.Deployment objects. It is meant to be used only in unit tests.
func NewDeployment() *DeploymentBuilder {
	return &DeploymentBuilder{
		deployment: appsv1.Deployment{},
	}
}

func (d *DeploymentBuilder) WithPodSpec(ps corev1.PodSpec) *DeploymentBuilder {
	d.deployment.Spec.Template.Spec = ps
	return d
}

func (d *DeploymentBuilder) WithReplicas(num *int32) *DeploymentBuilder {
	d.deployment.Spec.Replicas = num
	return d
}

func (d *DeploymentBuilder) WithSelectorAndPodLabels(kv ...string) *DeploymentBuilder {
	if d.deployment.Spec.Template.Labels == nil {
		d.deployment.Spec.Template.Labels = make(map[string]string)
	}
	if d.deployment.Spec.Selector == nil {
		d.deployment.Spec.Selector = &metav1.LabelSelector{}
	}
	if d.deployment.Spec.Selector.MatchLabels == nil {
		d.deployment.Spec.Selector.MatchLabels = make(map[string]string)
	}
	if len(kv)%2 != 0 {
		panic("the number of arguments must be even")
	}
	for i := 0; i < len(kv); i += 2 {
		d.deployment.Spec.Template.Labels[kv[i]] = kv[i+1]
		d.deployment.Spec.Selector.MatchLabels[kv[i]] = kv[i+1]
	}
	return d
}

func (d *DeploymentBuilder) WithName(name string) *DeploymentBuilder {
	d.deployment.Name = name
	return d
}

func (d *DeploymentBuilder) WithNamespace(namespace string) *DeploymentBuilder {
	d.deployment.Namespace = namespace
	return d
}

func (d *DeploymentBuilder) Build() appsv1.Deployment {
	return d.deployment
}
