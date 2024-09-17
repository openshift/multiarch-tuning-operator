/*
Copyright 2023 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package image

import (
	"context"
	"encoding/hex"
	"hash/fnv"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sync"

	"k8s.io/apimachinery/pkg/util/sets"
)

type cacheProxy struct {
	registryInspector        IRegistryInspector
	imageRefsArchitectureMap map[string]sets.Set[string]
	mutex                    sync.Mutex
}

func (c *cacheProxy) GetCompatibleArchitecturesSet(ctx context.Context, imageReference string, secrets [][]byte) (sets.Set[string], error) {
	authJson, err := marshaledImagePullSecrets(secrets)
	if err != nil {
		return nil, err
	}

	log := ctrllog.FromContext(ctx, "cacheProxy")

	if architectures, ok := c.imageRefsArchitectureMap[computeFNV128Hash(imageReference, authJson)]; ok {
		log.V(3).Info("Cache hit", "imageReference", imageReference)
		return architectures, nil
	}
	architectures, err := c.registryInspector.GetCompatibleArchitecturesSet(ctx, imageReference, secrets)
	if err != nil {
		return nil, err
	}

	log.V(3).Info("Cache miss...adding to cache", "imageReference", imageReference)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.imageRefsArchitectureMap[computeFNV128Hash(imageReference, authJson)] = architectures
	return architectures, nil
}

func (c *cacheProxy) GetRegistryInspector() IRegistryInspector {
	return c.registryInspector
}

func newCacheProxy() *cacheProxy {
	return &cacheProxy{
		imageRefsArchitectureMap: map[string]sets.Set[string]{},
		registryInspector:        newRegistryInspector(),
	}
}

func computeFNV128Hash(imageReference string, secrets []byte) string {
	hash := fnv.New128()
	hash.Write([]byte(imageReference)) // Add the image reference
	hash.Write(secrets)                // Add the secrets

	return hex.EncodeToString(hash.Sum(nil))
}

// TODO: eviction policy
