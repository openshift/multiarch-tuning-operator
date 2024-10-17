package builder

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DaemonSetBuilder is a builder for appsv1.DaemonSet objects to be used only in unit tests.
type DaemonSetBuilder struct {
	daemonset *appsv1.DaemonSet
}

// NewDaemonSet returns a new DaemonSetBuilder to build appsv1.DaemonSet objects. It is meant to be used only in unit tests.
func NewDaemonSet() *DaemonSetBuilder {
	return &DaemonSetBuilder{
		daemonset: &appsv1.DaemonSet{},
	}
}

func (d *DaemonSetBuilder) WithPodSpec(ps corev1.PodSpec) *DaemonSetBuilder {
	d.daemonset.Spec.Template.Spec = ps
	return d
}

func (d *DaemonSetBuilder) WithSelectorAndPodLabels(entries map[string]string) *DaemonSetBuilder {
	if d.daemonset.Spec.Template.Labels == nil && len(entries) > 0 {
		d.daemonset.Spec.Template.Labels = make(map[string]string, len(entries))
	}
	if d.daemonset.Spec.Selector == nil {
		d.daemonset.Spec.Selector = &metav1.LabelSelector{}
	}
	if d.daemonset.Spec.Selector.MatchLabels == nil && len(entries) > 0 {
		d.daemonset.Spec.Selector.MatchLabels = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		d.daemonset.Spec.Template.Labels[k] = v
		d.daemonset.Spec.Selector.MatchLabels[k] = v
	}
	return d
}

func (d *DaemonSetBuilder) WithName(name string) *DaemonSetBuilder {
	d.daemonset.Name = name
	return d
}

func (d *DaemonSetBuilder) WithNamespace(namespace string) *DaemonSetBuilder {
	d.daemonset.Namespace = namespace
	return d
}

func (d *DaemonSetBuilder) Build() *appsv1.DaemonSet {
	return d.daemonset
}
