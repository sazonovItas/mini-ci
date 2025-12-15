package runner

import (
	"context"
	"sync"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/workflower/db"
)

type TaskLogSaver struct {
	bus     events.Bus
	taskLog *db.TaskLogRepository

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func NewTaskLogSaver(bus events.Bus, taskLog *db.TaskLogRepository) *TaskLogSaver {
	return &TaskLogSaver{
		bus:     bus,
		taskLog: taskLog,
	}
}

func (r *TaskLogSaver) Start(ctx context.Context) error {
	r.ctx, r.cancel = context.WithCancel(ctx)

	r.wg.Go(func() { r.startSavingLogs(r.ctx) })

	return nil
}

func (r *TaskLogSaver) Stop(ctx context.Context) error {
	r.cancel()

	r.wg.Wait()

	return nil
}

func (r *TaskLogSaver) startSavingLogs(ctx context.Context) {
	evch, errs := r.bus.Subscribe(
		ctx,
		events.WithEventTypes(events.EventTypeTaskLog),
	)

	for {
		select {
		case <-ctx.Done():
			return

		case ev, ok := <-evch:
			if !ok {
				log.G(ctx).Debug("worker io bus channel is closed")
				return
			}

			event := ev.(events.TaskLog)

			if err := r.taskLog.Save(ctx, event.Origin().ID, event.Messages...); err != nil {
				log.G(ctx).WithError(err).Errorf("failed to save logs for the task %s", event.Origin().ID)
			}

		case err := <-errs:
			if err != nil {
				log.G(ctx).WithError(err).Error("failed to listen on bus")
			}
		}
	}
}
