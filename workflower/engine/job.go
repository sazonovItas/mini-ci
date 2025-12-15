package engine

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/workflower/db"
	"github.com/sazonovItas/mini-ci/workflower/model"
)

var ErrTaskNotFoundForSchedule = errors.New("task not found for schedule")

type JobWatcher struct {
	bus   events.Bus
	jobs  *db.JobFactory
	tasks *db.TaskFactory

	wg sync.WaitGroup

	ctx    context.Context
	cancel context.CancelFunc
}

func NewJobWatcher(bus events.Bus, jobs *db.JobFactory, tasks *db.TaskFactory) *JobWatcher {
	return &JobWatcher{
		bus:   bus,
		jobs:  jobs,
		tasks: tasks,
	}
}

func (w *JobWatcher) Start(ctx context.Context) {
	w.ctx, w.cancel = context.WithCancel(ctx)

	w.wg.Go(func() { w.watchEvents(w.ctx) })
}

func (w *JobWatcher) Stop(ctx context.Context) {
	w.cancel()

	w.wg.Wait()
}

func (w *JobWatcher) watchEvents(ctx context.Context) {
	evch, errch := w.bus.Subscribe(
		ctx,
		events.WithEventTypes(
			events.EventTypeJobStatus,
			events.EventTypeTaskStatus,
		),
	)

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-evch:
			if !ok {
				return
			}

			err := w.processEvent(ctx, event)
			if err != nil {
				log.G(ctx).WithError(err).Error("task watcher: failed to process event")
			}

		case err := <-errch:
			if err != nil {
				log.G(ctx).WithError(err).Error("job watcher: failed to read events from bus")
			}
		}
	}
}

func (w *JobWatcher) processEvent(ctx context.Context, ev events.Event) (err error) {
	switch event := ev.(type) {
	case events.JobStatus:
		err = w.pendingStatus(ctx, event)

	case events.TaskStatus:

	default:
		log.G(ctx).Errorf("job watcher: unknown event type %T to process", event)
	}

	return err
}

func (w *JobWatcher) pendingStatus(ctx context.Context, event events.JobStatus) error {
	if event.Status.IsPending() {
		return nil
	}

	job, found, err := w.jobs.Job(ctx, event.Origin().ID)
	if err != nil {
		return err
	}

	if !found {
		return nil
	}

	return job.WithTx(ctx, func(txCtx context.Context) error { return nil })
}

func (w *JobWatcher) scheduleNextTask(ctx context.Context, job db.Job) (stop bool, task db.Task, err error) {
	return
}

func (w *JobWatcher) publishStatusChanged(ctx context.Context, job model.Job) error {
	event := events.JobStatus{
		ChangeStatus: events.ChangeStatus{
			EventOrigin: events.EventOrigin{
				ID:        job.ID,
				OccuredAt: time.Now().UTC(),
			},
			Status: job.Status,
		},
		BuildID: job.BuildID,
	}

	return w.bus.Publish(ctx, event)
}
