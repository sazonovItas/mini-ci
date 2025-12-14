package runner

import (
	"context"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
)

type APIServerRunnerConfig struct {
	Address string
}

type APIServerRunner struct {
	bus events.Bus

	httpServer *HTTPServerRunner

	ctx    context.Context
	cancel context.CancelFunc
}

func NewAPIServerRunner(bus events.Bus, cfg APIServerRunnerConfig) *APIServerRunner {
	// TODO: add handler for api server
	httpServer := NewHTTPServerRunner(cfg.Address, nil)

	return &APIServerRunner{
		bus:        bus,
		httpServer: httpServer,
	}
}

func (r *APIServerRunner) Start(parentCtx context.Context) error {
	r.ctx, r.cancel = context.WithCancel(parentCtx)

	if err := r.httpServer.Start(r.ctx); err != nil {
		log.G(r.ctx).WithError(err).Error("failed to start api http server")
	}

	return nil
}

func (r *APIServerRunner) Stop(ctx context.Context) error {
	r.cancel()

	if err := r.httpServer.Stop(ctx); err != nil {
		log.G(ctx).WithError(err).Error("failed to stop api http server")
	}

	return nil
}
