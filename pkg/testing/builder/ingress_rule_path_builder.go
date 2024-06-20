package builder

import v1 "k8s.io/api/networking/v1"

// rulePathBuilder is a builder for v1.HTTPIngressPath objects to be used only in unit tests.
type rulePathBuilder struct {
	rulePath v1.HTTPIngressPath
}

// NewRulePath returns a new rulePathBuilder to build v1.HTTPIngressPath objects. It is meant to be used only in unit tests.
func NewRulePath() *rulePathBuilder {
	return &rulePathBuilder{
		rulePath: v1.HTTPIngressPath{},
	}
}

func (p *rulePathBuilder) WithRulePathPath(path string) *rulePathBuilder {
	p.rulePath.Path = path
	return p
}

func (p *rulePathBuilder) WithRulePathType(pathType v1.PathType) *rulePathBuilder {
	p.rulePath.PathType = &pathType
	return p
}

func (p *rulePathBuilder) WithRulePathPathTypePrefix() *rulePathBuilder {
	return p.WithRulePathType("Prefix")
}

func (p *rulePathBuilder) WithRulePathBackendService(name string, port int32) *rulePathBuilder {
	if p.rulePath.Backend.Service == nil {
		p.rulePath.Backend.Service = &v1.IngressServiceBackend{}
	}
	p.rulePath.Backend.Service.Name = name
	p.rulePath.Backend.Service.Port = v1.ServiceBackendPort{
		Number: port,
	}
	return p
}

func (p *rulePathBuilder) Build() v1.HTTPIngressPath {
	return p.rulePath
}
