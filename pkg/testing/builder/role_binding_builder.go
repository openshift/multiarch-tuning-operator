package builder

import rbacv1 "k8s.io/api/rbac/v1"

type RoleBindingBuilder struct {
	roleBinding *rbacv1.RoleBinding
}

// NewRoleBinding returns a new RoleBindingBuilder to build rbacv1.RoleBinding objects. It is meant to be used only in unit tests.
func NewRoleBinding() *RoleBindingBuilder {
	return &RoleBindingBuilder{
		roleBinding: &rbacv1.RoleBinding{},
	}
}

func (c *RoleBindingBuilder) Build() *rbacv1.RoleBinding {
	return c.roleBinding
}

func (c *RoleBindingBuilder) WithName(name string) *RoleBindingBuilder {
	c.roleBinding.Name = name
	return c
}

func (c *RoleBindingBuilder) WithNamespace(namespace string) *RoleBindingBuilder {
	c.roleBinding.Namespace = namespace
	return c
}
