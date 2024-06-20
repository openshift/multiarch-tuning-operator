package builder

import v1 "k8s.io/api/networking/v1"

// IngressRuleBuilder is a builder for v1.IngressRule objects to be used only in unit tests.
type IngressRuleBuilder struct {
	ingressRule v1.IngressRule
}

// NewIngressRule returns a new IngressRuleBuilder to build v1.IngressRule objects. It is meant to be used only in unit tests.
func NewIngressRule() *IngressRuleBuilder {
	return &IngressRuleBuilder{
		ingressRule: v1.IngressRule{},
	}
}

func (r *IngressRuleBuilder) WithIngressRuleHost(host string) *IngressRuleBuilder {
	r.ingressRule.Host = host
	return r
}

func (r *IngressRuleBuilder) WithIngressRuleHttpPath(paths ...v1.HTTPIngressPath) *IngressRuleBuilder {
	if r.ingressRule.HTTP == nil {
		r.ingressRule.HTTP = &v1.HTTPIngressRuleValue{}
	}
	r.ingressRule.HTTP.Paths = append(r.ingressRule.HTTP.Paths, paths...)
	return r
}

func (r *IngressRuleBuilder) Build() v1.IngressRule {
	return r.ingressRule
}
