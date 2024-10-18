package builder

import rbacv1 "k8s.io/api/rbac/v1"

// RoleBuilder is a builder for rbacv1.ClusterRole objects to be used only in unit tests.
type RoleBuilder struct {
	role *rbacv1.Role
}

// NewRole returns a new ClusterRoleBuilder to build rbacv1.ClusterRole objects. It is meant to be used only in unit tests.
func NewRole() *RoleBuilder {
	return &RoleBuilder{
		role: &rbacv1.Role{},
	}
}

func (c *RoleBuilder) Build() *rbacv1.Role {
	return c.role
}

func (c *RoleBuilder) WithName(name string) *RoleBuilder {
	c.role.Name = name
	return c
}

func (c *RoleBuilder) WithNamespace(namespace string) *RoleBuilder {
	c.role.Namespace = namespace
	return c
}
