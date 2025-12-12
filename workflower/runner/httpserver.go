package runner

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/containerd/log"
)

type HTTPServerRunner struct {
	server *http.Server
	wg     sync.WaitGroup
}

func NewHTTPServerRunner(address string, handler http.Handler) *HTTPServerRunner {
	server := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 250 * time.Millisecond,
	}

	return &HTTPServerRunner{
		server: server,
	}
}

func (r *HTTPServerRunner) Start(ctx context.Context) error {
	r.wg.Go(func() {
		if err := r.server.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.G(ctx).WithError(err).Error("failed to start http server")
			}
		}
	})

	return nil
}

func (r *HTTPServerRunner) Stop(ctx context.Context) error {
	if err := r.server.Shutdown(ctx); err != nil {
		log.G(ctx).WithError(err).Error("failed to shutdown http server")
	}

	r.wg.Wait()

	return nil
}
