package builder

import (
	v1 "k8s.io/api/core/v1"
)

// NodeSelectorTermBuilder is a builder for v1.NodeSelectorTerm objects to be used only in unit tests.
type NodeSelectorTermBuilder struct {
	nodeSelectorTerm *v1.NodeSelectorTerm
}

// NewNodeSelectorTerm returns a new NodeSelectorTermBuilder to build v1.NodeSelectorTerm objects. It is meant to be used only in unit tests.
func NewNodeSelectorTerm() *NodeSelectorTermBuilder {
	return &NodeSelectorTermBuilder{
		nodeSelectorTerm: &v1.NodeSelectorTerm{},
	}
}

func (n *NodeSelectorTermBuilder) WithMatchExpressions(values ...*v1.NodeSelectorRequirement) *NodeSelectorTermBuilder {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithMatchExpressions")
		}
		n.nodeSelectorTerm.MatchExpressions = append(n.nodeSelectorTerm.MatchExpressions, *values[i])
	}
	return n
}

func (n *NodeSelectorTermBuilder) WithMatchFields(values ...*v1.NodeSelectorRequirement) *NodeSelectorTermBuilder {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithMatchExpressions")
		}
		n.nodeSelectorTerm.MatchFields = append(n.nodeSelectorTerm.MatchFields, *values[i])
	}
	return n
}

func (n *NodeSelectorTermBuilder) Build() *v1.NodeSelectorTerm {
	return n.nodeSelectorTerm
}
