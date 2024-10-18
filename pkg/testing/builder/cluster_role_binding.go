package builder

import (
	v1 "k8s.io/api/rbac/v1"
)

// ClusterRoleBindingBuilder is a builder for v1.ClusterRoleBinding objects to be used only in unit tests.
type ClusterRoleBindingBuilder struct {
	clusterRoleBinding *v1.ClusterRoleBinding
}

// NewClusterRoleBinding returns a new ClusterRoleBindingBuilder to build v1.ClusterRoleBinding objects. It is meant to be used only in unit tests.
func NewClusterRoleBinding() *ClusterRoleBindingBuilder {
	return &ClusterRoleBindingBuilder{
		clusterRoleBinding: &v1.ClusterRoleBinding{},
	}
}

func (c *ClusterRoleBindingBuilder) WithRoleRef(apiGroup string, kind string, name string) *ClusterRoleBindingBuilder {
	c.clusterRoleBinding.RoleRef = v1.RoleRef{
		APIGroup: apiGroup,
		Kind:     kind,
		Name:     name,
	}
	return c
}

func (c *ClusterRoleBindingBuilder) WithName(name string) *ClusterRoleBindingBuilder {
	c.clusterRoleBinding.Name = name
	return c
}

func (c *ClusterRoleBindingBuilder) WithSubjects(subjects ...*v1.Subject) *ClusterRoleBindingBuilder {
	for i := range subjects {
		if subjects[i] == nil {
			panic("nil value passed to WithSubjects")
		}
		c.clusterRoleBinding.Subjects = append(c.clusterRoleBinding.Subjects, *subjects[i])
	}
	return c
}

func (c *ClusterRoleBindingBuilder) Build() *v1.ClusterRoleBinding {
	return c.clusterRoleBinding
}
