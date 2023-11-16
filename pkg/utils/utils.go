package utils

import (
	"sort"

	v1 "k8s.io/api/core/v1"
)

func NewPtr[T any](a T) *T {
	return &a
}

func SortMatchExpressions(nst []v1.NodeSelectorTerm) []v1.NodeSelectorTerm {
	for _, term := range nst {
		for _, req := range term.MatchExpressions {
			sort.Strings(req.Values)
		}
	}
	return nst
}
