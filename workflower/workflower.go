package workflower

import (
	"context"
	"sync"

	"github.com/containerd/log"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sazonovItas/mini-ci/core/events/exchange"
	"github.com/sazonovItas/mini-ci/workflower/config"
	pgdb "github.com/sazonovItas/mini-ci/workflower/db"
	"github.com/sazonovItas/mini-ci/workflower/runner"
)

type Workflower struct {
	workerIO  *runner.WorkerIORunner
	apiServer *runner.APIServerRunner

	db  *pgdb.DB
	bus *exchange.Exchange

	pgpool *pgxpool.Pool

	wg sync.WaitGroup
}

func New(cfg config.Config) (*Workflower, error) {
	pgpool, err := pgdb.Connect(context.TODO(), cfg.Postgres.URI)
	if err != nil {
		return nil, err
	}

	db := pgdb.New(pgpool)
	bus := exchange.NewExchange()

	workerIO := runner.NewWorkerIORunner(
		bus,
		runner.WorkerIORunnerConfig{
			Address:  cfg.WorkerIO.Addresss,
			Endpoint: cfg.WorkerIO.Endpoint,
		},
	)

	apiServer := runner.NewAPIServerRunner(
		bus,
		runner.APIServerRunnerConfig{
			Address: cfg.API.Address,
		},
	)

	workflower := &Workflower{
		apiServer: apiServer,
		workerIO:  workerIO,
		bus:       bus,
		db:        db,
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
	_ = w.bus.Close()
	w.pgpool.Close()
}
