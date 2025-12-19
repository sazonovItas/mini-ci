package runner

import (
	"context"

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

func (r *APIServer) Start(ctx context.Context) {
	r.ctx, r.cancel = context.WithCancel(ctx)

	r.httpServer.Start(r.ctx)
}

func (r *APIServer) Stop(ctx context.Context) {
	r.cancel()

	r.httpServer.Stop(ctx)
}
