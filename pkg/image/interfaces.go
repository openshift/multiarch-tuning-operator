package image

import (
	"context"
	"k8s.io/apimachinery/pkg/util/sets"
)

type ICache interface {
	// GetCompatibleArchitecturesSet takes an image reference. a list of secrets and the client to the cluster and
	// returns a set of architectures that are compatible with the image reference.
	GetCompatibleArchitecturesSet(ctx context.Context, imageReference string, secrets [][]byte) (sets.Set[string], error)
}

type IRegistryInspector interface {
	ICache
	// StoreGlobalPullSecret takes a pull secret and stores it in the ImageFacade. It will be used by the controller
	// in charge of watching the global pull secret and to store it in the ImageFacade's relevant private field.
	// Then, the ImageFacade will be responsible for consuming it during the inspection.
	StoreGlobalPullSecret(pullSecret []byte)
}
