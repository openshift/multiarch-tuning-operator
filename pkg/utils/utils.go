package utils

import (
	"path"
)

func NewPtr[T any](a T) *T {
	return &a
}

func ArchLabelValue(arch string) string {
	return path.Join(LabelGroup, arch)
}
