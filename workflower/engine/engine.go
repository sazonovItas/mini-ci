package engine

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/core/events/watcher"
	"github.com/sazonovItas/mini-ci/core/status"
	"github.com/sazonovItas/mini-ci/workflower/db"
)

var (
	ErrBuildNotFound = errors.New("build not found")
	ErrJobNotFound   = errors.New("job not found")
	ErrTaskNotFound  = errors.New("task not found")
)

type Engine struct {
	db  *db.DB
	bus events.Bus

	watchers []*watcher.Watcher

	wg sync.WaitGroup
}

func New(
	db *db.DB,
	bus events.Bus,
) *Engine {
	return &Engine{
		db:  db,
		bus: bus,

		watchers: []*watcher.Watcher{
			watcher.NewWatcher(
				bus,
				NewBuildProcessor(db.BuildFactory(), db.JobFactory(), bus),
				watcher.WithSyncProcessing(),
			),
			watcher.NewWatcher(
				bus,
				NewJobProcessor(db.JobFactory(), db.TaskFactory(), bus),
				watcher.WithSyncProcessing(),
			),
			watcher.NewWatcher(
				bus,
				NewTaskProcessor(db.TaskFactory(), bus),
				watcher.WithSyncProcessing(),
			),
		},
	}
}

func (e *Engine) Start(ctx context.Context) {
	for _, w := range e.watchers {
		w.Start(ctx)
	}

	e.wg.Go(func() { e.triggerBuildsSchedule(ctx) })
	e.wg.Go(func() { e.triggerJobsSchedule(ctx) })
}

func (e *Engine) Stop() {
	for _, w := range e.watchers {
		w.Stop()
	}

	e.wg.Wait()
}

func (e *Engine) triggerBuildsSchedule(ctx context.Context) {
	const (
		timeout = 5 * time.Second
	)

	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			builds, err := e.db.BuildRepository().BuildsForSchedule(ctx)
			if err != nil {
				log.G(ctx).WithError(err).Error("failed to reschedule builds")
				continue
			}

			for _, b := range builds {
				_ = e.bus.Publish(
					ctx,
					events.JobStatus{
						ChangeStatus: events.ChangeStatus{
							Status: status.StatusSucceeded,
						},
						BuildID: b.ID,
					},
				)
			}
		}
	}
}

func (e *Engine) triggerJobsSchedule(ctx context.Context) {
	const (
		timeout = 5 * time.Second
	)

	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			jobs, err := e.db.JobRepository().JobForSchedules(ctx)
			if err != nil {
				log.G(ctx).WithError(err).Error("failed to reschedule builds")
				continue
			}

			for _, j := range jobs {
				_ = e.bus.Publish(
					ctx,
					events.TaskStatus{
						ChangeStatus: events.ChangeStatus{
							Status: status.StatusSucceeded,
						},
						JobID: j.ID,
					},
				)
			}
		}
	}
}
