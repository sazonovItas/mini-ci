package runner

import (
	"context"
	"net/http"

	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/workflower/api"
	"github.com/sazonovItas/mini-ci/workflower/db"
)

type APIServerConfig struct {
	Address          string
	SocketIOEndpoint string
}

type APIServer struct {
	httpServer    *HTTPServer
	eventsGateway *EventsGateway

	ctx    context.Context
	cancel context.CancelFunc
}

func NewAPIServer(
	db *db.DB,
	bus events.Bus,
	cfg APIServerConfig,
) *APIServer {
	apiHandler := api.New(db, bus)
	eventsGateway := NewEventGateway(bus)

	mux := http.NewServeMux()

	apiHandler.RegisterRoutes(mux)

	mux.Handle(cfg.SocketIOEndpoint, eventsGateway.Handler())

	httpServer := NewHTTPServer(cfg.Address, mux)

	return &APIServer{
		httpServer:    httpServer,
		eventsGateway: eventsGateway,
	}
}

func (r *APIServer) Start(ctx context.Context) {
	r.ctx, r.cancel = context.WithCancel(ctx)

	r.eventsGateway.Start(r.ctx)

	r.httpServer.Start(r.ctx)
}

func (r *APIServer) Stop(ctx context.Context) {
	r.cancel()

	r.httpServer.Stop(ctx)
	r.eventsGateway.Stop(ctx)
}
