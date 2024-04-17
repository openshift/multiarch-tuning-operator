package builder

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

// Jobbuilder is a builder for batchv1.Job objects to be used only in unit tests.
type Jobbuilder struct {
	job batchv1.Job
}

// NewJob returns a new Jobbuilder to build batchv1.Job objects. It is meant to be used only in unit tests.
func NewJob() *Jobbuilder {
	return &Jobbuilder{
		job: batchv1.Job{},
	}
}

func (j *Jobbuilder) WithPodSpec(ps corev1.PodSpec) *Jobbuilder {
	j.job.Spec.Template.Spec = ps
	return j
}

func (j *Jobbuilder) WithPodLabels(entries map[string]string) *Jobbuilder {
	if j.job.Spec.Template.Labels == nil && len(entries) > 0 {
		j.job.Spec.Template.Labels = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		j.job.Spec.Template.Labels[k] = v
	}
	return j
}

func (j *Jobbuilder) WithName(name string) *Jobbuilder {
	j.job.Name = name
	return j
}

func (j *Jobbuilder) WithNamespace(namespace string) *Jobbuilder {
	j.job.Namespace = namespace
	return j
}

func (j *Jobbuilder) Build() batchv1.Job {
	return j.job
}
