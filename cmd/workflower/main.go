package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/configutil"
	"github.com/sazonovItas/mini-ci/workflower"
	"github.com/sazonovItas/mini-ci/workflower/config"
)

const (
	envPrefix = "MINICI"
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

	workflower, err := workflower.New(cfg)
	if err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	workflower.Start(ctx)

	<-ctx.Done()

	workflower.Stop(context.TODO())
}
