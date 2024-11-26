package informers

import (
	"sync"

	"github.com/openshift/multiarch-tuning-operator/apis/multiarch/v1beta1"
)

var (
	singletonSystemConfigInstance ICache
	once                          sync.Once
)

type ClusterPodPlacementConfig struct {
	config *v1beta1.ClusterPodPlacementConfig
	mu     sync.Mutex // Mutex for `config`
}

func CacheSingleton() ICache {
	once.Do(func() {
		singletonSystemConfigInstance = newCache()
	})
	return singletonSystemConfigInstance
}

func newCache() ICache {
	c := &ClusterPodPlacementConfig{
		config: nil,
	}
	return c
}

func (c *ClusterPodPlacementConfig) StoreClusterPodPlacementConfig(config *v1beta1.ClusterPodPlacementConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config = config.DeepCopy()
	return nil
}

func (c *ClusterPodPlacementConfig) DeleteClusterPodPlacementConfig() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config = nil
	return nil
}

func (c *ClusterPodPlacementConfig) GetClusterPodPlacementConfig() *v1beta1.ClusterPodPlacementConfig {
	return c.config
}
