package builder

import (
	v1 "k8s.io/api/core/v1"
)

// NodeSelectorRequirementBuilder is a builder for v1.NodeSelectorRequirement objects to be used only in unit tests.
type NodeSelectorRequirementBuilder struct {
	nodeSelectorRequirement v1.NodeSelectorRequirement
}

// NewNodeSelectorRequirement returns a new NodeSelectorRequirementBuilder to build v1.NodeSelectorRequirement objects. It is meant to be used only in unit tests.
func NewNodeSelectorRequirement() *NodeSelectorRequirementBuilder {
	return &NodeSelectorRequirementBuilder{
		nodeSelectorRequirement: v1.NodeSelectorRequirement{},
	}
}

func (n *NodeSelectorRequirementBuilder) WithKeyAndValues(key string, operator v1.NodeSelectorOperator, values ...string) *NodeSelectorRequirementBuilder {
	n.nodeSelectorRequirement.Key = key
	n.nodeSelectorRequirement.Operator = operator
	n.nodeSelectorRequirement.Values = values
	return n
}

func (n *NodeSelectorRequirementBuilder) Build() v1.NodeSelectorRequirement {
	return n.nodeSelectorRequirement
}
