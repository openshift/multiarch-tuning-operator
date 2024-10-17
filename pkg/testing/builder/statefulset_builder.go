package builder

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StatefulSetBuilder is a builder for appsv1.StatefulSet objects to be used only in unit tests.
type StatefulSetBuilder struct {
	statefulset *appsv1.StatefulSet
}

// NewStatefulSet returns a new StatefulSetBuilder to build appsv1.StatefulSet objects. It is meant to be used only in unit tests.
func NewStatefulSet() *StatefulSetBuilder {
	return &StatefulSetBuilder{
		statefulset: &appsv1.StatefulSet{},
	}
}

func (s *StatefulSetBuilder) WithPodSpec(ps corev1.PodSpec) *StatefulSetBuilder {
	s.statefulset.Spec.Template.Spec = ps
	return s
}

func (s *StatefulSetBuilder) WithReplicas(num *int32) *StatefulSetBuilder {
	s.statefulset.Spec.Replicas = num
	return s
}

func (s *StatefulSetBuilder) WithSelectorAndPodLabels(entries map[string]string) *StatefulSetBuilder {
	if s.statefulset.Spec.Template.Labels == nil && len(entries) > 0 {
		s.statefulset.Spec.Template.Labels = make(map[string]string, len(entries))
	}
	if s.statefulset.Spec.Selector == nil {
		s.statefulset.Spec.Selector = &metav1.LabelSelector{}
	}
	if s.statefulset.Spec.Selector.MatchLabels == nil && len(entries) > 0 {
		s.statefulset.Spec.Selector.MatchLabels = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		s.statefulset.Spec.Template.Labels[k] = v
		s.statefulset.Spec.Selector.MatchLabels[k] = v
	}
	return s
}

func (s *StatefulSetBuilder) WithName(name string) *StatefulSetBuilder {
	s.statefulset.Name = name
	return s
}

func (s *StatefulSetBuilder) WithNamespace(namespace string) *StatefulSetBuilder {
	s.statefulset.Namespace = namespace
	return s
}

func (s *StatefulSetBuilder) Build() *appsv1.StatefulSet {
	return s.statefulset
}
