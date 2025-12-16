package runner

import (
	"context"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/core/events/watcher"
	"github.com/sazonovItas/mini-ci/workflower/db"
)

type TaskLogSaver struct {
	watcher *watcher.Watcher
}

func NewTaskLogSaver(subscriber events.Subscriber, taskLog *db.TaskLogRepository) TaskLogSaver {
	return TaskLogSaver{
		watcher: watcher.NewWatcher(
			subscriber,
			NewTaskLogProcessor(taskLog),
		),
	}
}

func (s TaskLogSaver) Start(ctx context.Context) {
	s.watcher.Start(ctx)
}

func (s TaskLogSaver) Stop() {
	s.watcher.Stop()
}

type TaskLogProcessor struct {
	log *db.TaskLogRepository
}

func NewTaskLogProcessor(taskLog *db.TaskLogRepository) TaskLogProcessor {
	return TaskLogProcessor{log: taskLog}
}

func (s TaskLogProcessor) Filters() []events.FilterFunc {
	return []events.FilterFunc{
		events.WithEventTypes(events.EventTypeTaskLog),
	}
}

func (s TaskLogProcessor) ProcessEvent(ctx context.Context, event events.Event) error {
	logEvent := event.(events.TaskLog)

	err := s.log.Save(ctx, event.Origin().ID, logEvent.Messages...)
	if err != nil {
		log.G(ctx).WithError(err).Errorf("failed to save logs for the task %s", event.Origin().ID)
		return err
	}

	return nil
}
