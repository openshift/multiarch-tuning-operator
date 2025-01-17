package builder

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OwnerReferenceBuilder is a builder for metav1.OwnerReference objects to be used only in unit tests.
type OwnerReferenceBuilder struct {
	ownerReference *metav1.OwnerReference
}

// NewOwnerReferenceBuilder returns a new OwnerReferenceBuilder to build metav1.OwnerReference objects. It is meant to be used only in unit tests.
func NewOwnerReferenceBuilder() *OwnerReferenceBuilder {
	return &OwnerReferenceBuilder{
		ownerReference: &metav1.OwnerReference{},
	}
}

func (o *OwnerReferenceBuilder) WithKind(kind string) *OwnerReferenceBuilder {
	o.ownerReference.Kind = kind
	return o
}

func (o *OwnerReferenceBuilder) WithController(controller *bool) *OwnerReferenceBuilder {
	o.ownerReference.Controller = controller
	return o
}

func (o *OwnerReferenceBuilder) Build() *metav1.OwnerReference {
	return o.ownerReference
}
