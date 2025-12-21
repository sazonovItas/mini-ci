package workflower

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sazonovItas/mini-ci/core/events/exchange"
	"github.com/sazonovItas/mini-ci/workflower/config"
	pgdb "github.com/sazonovItas/mini-ci/workflower/db"
	"github.com/sazonovItas/mini-ci/workflower/engine"
	"github.com/sazonovItas/mini-ci/workflower/runner"
)

type Workflower struct {
	workerIO   *runner.WorkerIO
	apiServer  *runner.APIServer
	logSaver   *runner.TaskLogSaver
	eventSaver *runner.EventSaver

	engine *engine.Engine

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

	workerIO := runner.NewWorkerIO(
		bus,
		runner.WorkerIOConfig{
			Address:  cfg.WorkerIO.Addresss,
			Endpoint: cfg.WorkerIO.Endpoint,
		},
	)

	apiServer := runner.NewAPIServer(
		db,
		bus,
		runner.APIServerConfig{
			Address:          cfg.API.Address,
			SocketIOEndpoint: cfg.API.SocketIOEndpoint,
		},
	)

	logSaver := runner.NewTaskLogSaver(bus, db.TaskLogRepository())

	eventSaver := runner.NewEventSaver(bus, db.EventRepository())

	engine := engine.New(db, bus)

	workflower := &Workflower{
		apiServer:  apiServer,
		workerIO:   workerIO,
		logSaver:   logSaver,
		eventSaver: eventSaver,
		engine:     engine,

		bus:    bus,
		db:     db,
		pgpool: pgpool,
	}

	return workflower, nil
}

func (w *Workflower) Start(ctx context.Context) {
	w.wg.Go(func() { w.engine.Start(ctx) })
	w.wg.Go(func() { w.logSaver.Start(ctx) })
	w.wg.Go(func() { w.eventSaver.Start(ctx) })
	w.wg.Go(func() { w.workerIO.Start(ctx) })
	w.wg.Go(func() { w.apiServer.Start(ctx) })
}

func (w *Workflower) Stop(ctx context.Context) {
	w.wg.Go(func() { w.apiServer.Stop(ctx) })
	w.wg.Go(func() { w.workerIO.Stop(ctx) })
	w.wg.Go(func() { w.logSaver.Stop(ctx) })
	w.wg.Go(func() { w.eventSaver.Stop(ctx) })
	w.wg.Go(func() { w.engine.Stop() })

	w.wg.Wait()

	w.cleanup()
}

func (w *Workflower) cleanup() {
	_ = w.bus.Close()
	w.pgpool.Close()
}
