package builder

import rbacv1 "k8s.io/api/rbac/v1"

// ClusterRoleBuilder is a builder for rbacv1.ClusterRole objects to be used only in unit tests.
type ClusterRoleBuilder struct {
	clusterRoleBinding *rbacv1.ClusterRole
}

// NewClusterRole returns a new ClusterRoleBuilder to build rbacv1.ClusterRole objects. It is meant to be used only in unit tests.
func NewClusterRole() *ClusterRoleBuilder {
	return &ClusterRoleBuilder{
		clusterRoleBinding: &rbacv1.ClusterRole{},
	}
}

func (c *ClusterRoleBuilder) Build() *rbacv1.ClusterRole {
	return c.clusterRoleBinding
}

func (c *ClusterRoleBuilder) WithName(name string) *ClusterRoleBuilder {
	c.clusterRoleBinding.Name = name
	return c
}
