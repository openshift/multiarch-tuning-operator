package builder

import v1 "k8s.io/api/networking/v1"

// IngressBuilder is a builder for v1.Ingress objects to be used only in unit tests.
type IngressBuilder struct {
	ingress v1.Ingress
}

// NewIngress returns a new IngressBuilder to build v1.Ingress objects. It is meant to be used only in unit tests.
func NewIngress() *IngressBuilder {
	return &IngressBuilder{
		ingress: v1.Ingress{},
	}
}

func (i *IngressBuilder) WithName(name string) *IngressBuilder {
	i.ingress.Name = name
	return i
}

func (i *IngressBuilder) WithNamespace(namespace string) *IngressBuilder {
	i.ingress.Namespace = namespace
	return i
}

func (i *IngressBuilder) WithAnnotations(entries map[string]string) *IngressBuilder {
	if i.ingress.Annotations == nil && len(entries) > 0 {
		i.ingress.Annotations = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		i.ingress.Annotations[k] = v
	}
	return i
}

func (i *IngressBuilder) WithIngressRules(values ...v1.IngressRule) *IngressBuilder {
	i.ingress.Spec.Rules = append(i.ingress.Spec.Rules, values...)
	return i
}

func (i *IngressBuilder) Build() v1.Ingress {
	return i.ingress
}
