package workflower

import (
	"context"
	"sync"

	"github.com/containerd/log"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sazonovItas/mini-ci/workflower/config"
	"github.com/sazonovItas/mini-ci/workflower/db"
	"github.com/sazonovItas/mini-ci/workflower/runner"
)

type Workflower struct {
	workerIO  *runner.WorkerIORunner
	apiServer *runner.HTTPServerRunner

	pgpool *pgxpool.Pool

	wg sync.WaitGroup
}

func New(cfg config.Config) (*Workflower, error) {
	pgpool, err := db.Connect(context.TODO(), cfg.Postgres.URI)
	if err != nil {
		return nil, err
	}

	workerIO := runner.NewWorkerIORunner(cfg.WorkerIO.Addresss, cfg.WorkerIO.Endpoint)
	apiServer := runner.NewHTTPServerRunner(cfg.API.Address, nil)

	workflower := &Workflower{
		apiServer: apiServer,
		workerIO:  workerIO,
		pgpool:    pgpool,
	}

	return workflower, nil
}

func (w *Workflower) Start(ctx context.Context) error {
	w.wg.Go(func() {
		if err := w.workerIO.Start(ctx); err != nil {
			log.G(ctx).WithError(err).Error("failed to start worker socket io")
		}
	})

	w.wg.Go(func() {
		if err := w.apiServer.Start(ctx); err != nil {
			log.G(ctx).WithError(err).Error("failed to start api server")
		}
	})

	return nil
}

func (w *Workflower) Stop(ctx context.Context) error {
	w.wg.Go(func() {
		if err := w.workerIO.Stop(ctx); err != nil {
			log.G(ctx).WithError(err).Error("failed to stop worker socket io")
		}
	})

	w.wg.Go(func() {
		if err := w.apiServer.Stop(ctx); err != nil {
			log.G(ctx).WithError(err).Error("failed to stop api server")
		}
	})

	w.wg.Wait()

	w.cleanup()

	return nil
}

func (w *Workflower) cleanup() {
	w.pgpool.Close()
}
