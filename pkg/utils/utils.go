package utils

import "path"

func NewPtr[T any](a T) *T {
	return &a
}

func HasControlPlaneNodeSelector(nodeSelector map[string]string) bool {
	requiredSelectors := []string{MasterNodeSelectorLabel, ControlPlaneNodeSelectorLabel}

	for _, value := range requiredSelectors {
		if _, ok := nodeSelector[value]; ok {
			return true
		}
	}
	return false
}

func ArchLabelValue(arch string) string {
	return path.Join(LabelGroup, arch)
}
