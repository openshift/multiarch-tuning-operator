package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/time/rate"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/outrigger-project/multiarch-tuning-operator/enoexec-daemon/internal/storage"
	"github.com/outrigger-project/multiarch-tuning-operator/enoexec-daemon/internal/types"
)

var (
	initialLogLevel int
	logDevMode      bool
)

func main() {
	bindFlags()
	ctx, cancel := initContext()
	log, err := logr.FromContext(ctx)
	must(err, "failed to get logger from context")

	ch := make(chan *types.ENOEXECInternalEvent, 256)
	storageImpl, err := storage.NewK8sENOExecEventStorage(ctx,
		rate.NewLimiter(5, 10), ch, os.Getenv("NODE_NAME"), os.Getenv("NAMESPACE"), time.Minute,
	)
	must(err, "failed to create storage")

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
}

func bindFlags() {
	flag.IntVar(&initialLogLevel, "initial-log-level", 0, "Initial log level. From 0 (Normal) to 5 (TraceAll)")
	flag.BoolVar(&logDevMode, "log-dev-mode", false, "Enable development mode for zap logger")
	flag.Parse()
}

func initContext() (context.Context, context.CancelFunc) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	var logImpl *zap.Logger
	var err error
	if logDevMode {
		logImpl, err = zap.NewDevelopment()
	} else {
		level := zap.NewAtomicLevelAt(zapcore.Level(-initialLogLevel))
		logImpl, err = zap.NewProduction(zap.IncreaseLevel(level))
	}
	must(err, "failed to create logger")
	ctx = logr.NewContext(ctx, zapr.NewLogger(logImpl))
	return ctx, cancel
}

func must(err error, msg string, fns ...func()) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	for _, fn := range fns {
		fn()
	}
	os.Exit(1)
}
