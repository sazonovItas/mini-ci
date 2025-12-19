package runner

import (
	"context"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/core/events/watcher"
	"github.com/sazonovItas/mini-ci/workflower/db"
)

type EventSaver struct {
	watcher *watcher.Watcher
}

func NewEventSaver(subscriber events.Subscriber, eventRepo *db.EventRepository) *EventSaver {
	return &EventSaver{
		watcher: watcher.NewWatcher(
			subscriber,
			NewEventSaverProcessor(eventRepo),
			watcher.WithSyncProcessing(),
		),
	}
}

func (s EventSaver) Start(ctx context.Context) {
	s.watcher.Start(ctx)
}

func (s EventSaver) Stop(_ context.Context) {
	s.watcher.Stop()
}

type EventSaverProcessor struct {
	event *db.EventRepository
}

func NewEventSaverProcessor(eventRepo *db.EventRepository) EventSaverProcessor {
	return EventSaverProcessor{event: eventRepo}
}

func (p EventSaverProcessor) Filters() []events.FilterFunc {
	return nil
}

func (p EventSaverProcessor) ProcessEvent(ctx context.Context, event events.Event) error {
	err := p.event.Save(ctx, event)
	if err != nil {
		log.G(ctx).WithError(err).Errorf("failed to save event for the task %s", event.Origin().ID)
		return err
	}

	return nil
}
