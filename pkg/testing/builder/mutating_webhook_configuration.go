package builder

import v1 "k8s.io/api/admissionregistration/v1"

type MutatingWebhookConfigurationBuilder struct {
	mutatingWebhookConfiguration *v1.MutatingWebhookConfiguration
}

func NewMutatingWebhookConfiguration() *MutatingWebhookConfigurationBuilder {
	return &MutatingWebhookConfigurationBuilder{
		mutatingWebhookConfiguration: &v1.MutatingWebhookConfiguration{},
	}
}

func (m *MutatingWebhookConfigurationBuilder) WithName(name string) *MutatingWebhookConfigurationBuilder {
	m.mutatingWebhookConfiguration.Name = name
	return m
}

func (m *MutatingWebhookConfigurationBuilder) Build() *v1.MutatingWebhookConfiguration {
	return m.mutatingWebhookConfiguration
}
