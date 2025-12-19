package runner

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/containerd/log"
)

type HTTPServer struct {
	server *http.Server
	wg     sync.WaitGroup
}

func NewHTTPServer(address string, handler http.Handler) *HTTPServer {
	server := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 250 * time.Millisecond,
	}

	return &HTTPServer{
		server: server,
	}
}

func (r *HTTPServer) Start(ctx context.Context) {
	r.wg.Go(func() {
		if err := r.server.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.G(ctx).WithError(err).Error("failed to start http server")
			}
		}
	})
}

func (r *HTTPServer) Stop(ctx context.Context) {
	if err := r.server.Shutdown(ctx); err != nil {
		log.G(ctx).WithError(err).Error("failed to shutdown http server")
	}

	r.wg.Wait()
}
