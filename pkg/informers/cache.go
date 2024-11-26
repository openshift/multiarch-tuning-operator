package informers

import (
	"encoding/json"

	"sync"
)

var (
	singletonSystemConfigInstance ICache
	once                          sync.Once
	//log                           logr.Logger
)

type ClusterPodPlacementConfigSyncer struct {
	config        json.RawMessage // For raw JSON object
	webhookConfig json.RawMessage // For raw JSON object

	ch chan bool
	mu sync.Mutex
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
		webhookConfig: json.RawMessage{}, //Wrong type

		ch: make(chan bool),
	}
	return c
}

func (c *ClusterPodPlacementConfigSyncer) unlockAndSync() {
	c.mu.Unlock()
	c.ch <- true
}

func (c *ClusterPodPlacementConfigSyncer) StoreClusterPodPlacementConfig(CPPCconfig json.RawMessage, webhookConfig json.RawMessage) error {
	c.mu.Lock()
	defer c.unlockAndSync()
	c.config = CPPCconfig
	c.webhookConfig = webhookConfig
	return nil
}

func (c *ClusterPodPlacementConfigSyncer) DeleteClusterPodPlacementConfig() {
	c.mu.Lock()
	defer c.unlockAndSync()
	c.config = json.RawMessage{}
	c.webhookConfig = json.RawMessage{}
}
