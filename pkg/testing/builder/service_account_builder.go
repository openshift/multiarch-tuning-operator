package builder

import v1 "k8s.io/api/core/v1"

type ServiceAccountBuilder struct {
	serviceAccount *v1.ServiceAccount
}

// NewServiceAccount returns a new ServiceAccountBuilder to build v1.ServiceAccount objects. It is meant to be used only in unit tests.
func NewServiceAccount() *ServiceAccountBuilder {
	return &ServiceAccountBuilder{
		serviceAccount: &v1.ServiceAccount{},
	}
}

func (c *ServiceAccountBuilder) Build() *v1.ServiceAccount {
	return c.serviceAccount
}

func (c *ServiceAccountBuilder) WithName(name string) *ServiceAccountBuilder {
	c.serviceAccount.Name = name
	return c
}

func (c *ServiceAccountBuilder) WithNamespace(namespace string) *ServiceAccountBuilder {
	c.serviceAccount.Namespace = namespace
	return c
}
