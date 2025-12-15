package runner

import (
	"context"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
)

type APIServerConfig struct {
	Address string
}

type APIServer struct {
	bus events.Bus

	httpServer *HTTPServer

	ctx    context.Context
	cancel context.CancelFunc
}

func NewAPIServer(bus events.Bus, cfg APIServerConfig) *APIServer {
	// TODO: add handler for api server
	httpServer := NewHTTPServer(cfg.Address, nil)

	return &APIServer{
		bus:        bus,
		httpServer: httpServer,
	}
}

func (r *APIServer) Start(parentCtx context.Context) error {
	r.ctx, r.cancel = context.WithCancel(parentCtx)

	if err := r.httpServer.Start(r.ctx); err != nil {
		log.G(r.ctx).WithError(err).Error("failed to start api http server")
	}

	return nil
}

func (r *APIServer) Stop(ctx context.Context) error {
	r.cancel()

	if err := r.httpServer.Stop(ctx); err != nil {
		log.G(ctx).WithError(err).Error("failed to stop api http server")
	}

	return nil
}
