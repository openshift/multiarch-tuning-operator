package daemon

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"golang.org/x/time/rate"

	"github.com/openshift/multiarch-tuning-operator/controllers/enoexecevent/daemon/internal/storage"
	"github.com/openshift/multiarch-tuning-operator/controllers/enoexecevent/daemon/internal/types"
)

func RunDaemon(ctx context.Context, cancel context.CancelFunc) error {
	log, err := logr.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get logger from context: %w", err)
	}
	ch := make(chan *types.ENOEXECInternalEvent, 256)
	storageImpl, err := storage.NewK8sENOExecEventStorage(ctx,
		rate.NewLimiter(5, 10), ch, os.Getenv("NODE_NAME"), os.Getenv("NAMESPACE"), time.Minute,
	)
	if err != nil {
		return fmt.Errorf("failed to create K8sENOExecEventStorage: %w", err)
	}

	log.Info("Starting workers")
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		log.Info("Starting storage writer")
		defer wg.Done()
		defer cancel()
		err = storageImpl.Run()
		<-ctx.Done()
	}()
	wg.Add(1)
	go func() {
		log.Info("Starting ENOEXEC eBPF tracepoint")
		defer wg.Done()
		defer cancel()
		// TODO: Replace with actual tracepoint implementation
		<-ctx.Done()
	}()
	wg.Add(1)
	go func() {
		log.Info("Starting main loop")
		defer wg.Done()
		defer cancel()
		// TODO: Replace with actual event processing implementation
		if err = storageImpl.Store(nil); err != nil {
			log.Error(err, "Failed to store event")
			return
		}
		<-ctx.Done()
	}()

	log.Info("Controller started, waiting for events")
	<-ctx.Done()
	log.Info("Context channel closed. Waiting for the workers to terminate")
	wg.Wait()
	log.Info("All workers terminated, exiting")
	return nil
}
