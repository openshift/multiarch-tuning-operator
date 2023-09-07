package image

import (
	"context"
	"k8s.io/apimachinery/pkg/util/sets"
	"sync"
)

var (
	singletonImageFacade *Facade
	// once is used for lazy initialization of the singletonImageFacade
	once sync.Once
)

type Facade struct {
	inspectionCache ICache
}

func (i *Facade) GetCompatibleArchitecturesSet(ctx context.Context, imageReference string, secrets [][]byte) (architectures sets.Set[string], err error) {
	return i.inspectionCache.GetCompatibleArchitecturesSet(ctx, imageReference, secrets)
}

func newImageFacade() *Facade {
	return &Facade{
		inspectionCache: newCache(),
	}
}

func FacadeSingleton() *Facade {
	once.Do(func() {
		singletonImageFacade = newImageFacade()
	})
	return singletonImageFacade
}

func (i *Facade) StoreGlobalPullSecret(pullSecret []byte) {
	i.inspectionCache.(*cacheProxy).registryInspector.StoreGlobalPullSecret(pullSecret)
}
