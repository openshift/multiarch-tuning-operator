package podplacement

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/openshift/multiarch-manager-operator/pkg/systemconfig"
)

type ConfigSyncerRunnable struct {
	log logr.Logger
}

func NewConfigSyncerRunnable() *ConfigSyncerRunnable {
	return &ConfigSyncerRunnable{}
}

func (s *ConfigSyncerRunnable) Start(ctx context.Context) error {
	s.log = log.FromContext(ctx, "handler", "ConfigSyncerRunnable")
	s.log.Info("Starting System Config Syncer Consumer")
	return systemconfig.SystemConfigSyncerSingleton().Run(ctx)
}
