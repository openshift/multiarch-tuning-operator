package builder

import (
	"github.com/openshift/multiarch-tuning-operator/apis/multiarch/common"
	"github.com/openshift/multiarch-tuning-operator/apis/multiarch/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterPodPlacementConfigBuilder struct {
	*v1beta1.ClusterPodPlacementConfig
}

func NewClusterPodPlacementConfig() *ClusterPodPlacementConfigBuilder {
	return &ClusterPodPlacementConfigBuilder{
		ClusterPodPlacementConfig: &v1beta1.ClusterPodPlacementConfig{},
	}
}

func (p *ClusterPodPlacementConfigBuilder) WithName(name string) *ClusterPodPlacementConfigBuilder {
	p.Name = name
	return p
}

func (p *ClusterPodPlacementConfigBuilder) WithNamespaceSelector(labelSelector *v1.LabelSelector) *ClusterPodPlacementConfigBuilder {
	p.Spec.NamespaceSelector = labelSelector
	return p
}

func (p *ClusterPodPlacementConfigBuilder) WithLogVerbosity(logVerbosity common.LogVerbosityLevel) *ClusterPodPlacementConfigBuilder {
	p.Spec.LogVerbosity = logVerbosity
	return p
}

func (p *ClusterPodPlacementConfigBuilder) Build() *v1beta1.ClusterPodPlacementConfig {
	return p.ClusterPodPlacementConfig
}
