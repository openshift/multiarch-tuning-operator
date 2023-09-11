package image

import (
	"context"
	"k8s.io/apimachinery/pkg/util/sets"
	"sync"
)

type cacheProxy struct {
	registryInspector        IRegistryInspector
	imageRefsArchitectureMap map[string]sets.Set[string]
	mutex                    sync.Mutex
}

func (c *cacheProxy) GetCompatibleArchitecturesSet(ctx context.Context, imageReference string, secrets [][]byte) (sets.Set[string], error) {
	if c.imageRefsArchitectureMap[imageReference] != nil {
		return c.imageRefsArchitectureMap[imageReference], nil
	}
	architectures, err := c.registryInspector.GetCompatibleArchitecturesSet(ctx, imageReference, secrets)
	if err != nil {
		return nil, err
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.imageRefsArchitectureMap[imageReference] = architectures
	return architectures, nil
}

func newCache() ICache {
	return &cacheProxy{
		imageRefsArchitectureMap: map[string]sets.Set[string]{},
		registryInspector:        newRegistryInspector(),
	}
}

// TODO: eviction policy
