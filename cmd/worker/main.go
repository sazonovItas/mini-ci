package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/configutil"
	"github.com/sazonovItas/mini-ci/worker"
	"github.com/sazonovItas/mini-ci/worker/config"
)

const (
	envPrefix = "MINICI_WORKER"
)

func init() {
	_ = log.SetFormat(log.TextFormat)
	_ = log.SetLevel(log.DebugLevel.String())
}

func main() {
	var cfg config.Config
	err := configutil.Load(
		&cfg,
		config.DefaultConfig,
		configutil.WithConfigType("yaml"),
		configutil.WithEnvs(envPrefix),
	)
	if err != nil {
		panic(err)
	}

	log.G(context.TODO()).Debug(cfg)

	worker, err := worker.New(cfg)
	if err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT,
	)
	defer stop()

	if err := worker.Start(ctx); err != nil {
		panic(err)
	}

	<-ctx.Done()

	if err := worker.Stop(context.TODO()); err != nil {
		panic(err)
	}
}
