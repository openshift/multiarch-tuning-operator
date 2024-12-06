package informers

import (
	"encoding/json"
	"sync"
)

var (
	singletonSystemConfigInstance ICache
	once                          sync.Once
)

type ClusterPodPlacementConfigSyncer struct {
	config        json.RawMessage // For raw JSON object
	webhookConfig json.RawMessage // For raw JSON object

	configMu sync.Mutex // Mutex for `config`
}

func CacheSingleton() ICache {
	once.Do(func() {
		singletonSystemConfigInstance = newCache()
	})
	return singletonSystemConfigInstance
}

func newCache() ICache {
	c := &ClusterPodPlacementConfigSyncer{
		config:        json.RawMessage{},
		webhookConfig: json.RawMessage{},
	}
	return c
}

func (c *ClusterPodPlacementConfigSyncer) StoreClusterPodPlacementConfig(config json.RawMessage, webhookConfig json.RawMessage) error {
	c.configMu.Lock()
	defer c.configMu.Unlock()
	c.config = config
	c.webhookConfig = webhookConfig
	return nil
}

func (c *ClusterPodPlacementConfigSyncer) DeleteClusterPodPlacementConfig() error {
	c.configMu.Lock()
	defer c.configMu.Unlock()
	c.config = json.RawMessage{}
	c.webhookConfig = json.RawMessage{}
	return nil
}
