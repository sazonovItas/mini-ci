package watcher

import (
	"context"
	"sync"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
)

type WatcherOpt func(*Watcher)

type EventProcessor interface {
	Filters() []events.FilterFunc
	ProcessEvent(ctx context.Context, event events.Event) error
}

type Watcher struct {
	subscriber events.Subscriber
	processor  EventProcessor

	wg     sync.WaitGroup
	cancel context.CancelFunc

	sync bool
}

func WithSyncProcessing() WatcherOpt {
	return func(w *Watcher) {
		w.sync = true
	}
}

func NewWatcher(
	subscriber events.Subscriber,
	processor EventProcessor,
	opts ...WatcherOpt,
) *Watcher {
	return &Watcher{
		subscriber: subscriber,
		processor:  processor,
	}
}

func (w *Watcher) Start(ctx context.Context) {
	watchCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	w.wg.Go(func() { w.watch(watchCtx) })
}

func (w *Watcher) Stop() {
	w.cancel()

	w.wg.Wait()
}

func (w *Watcher) watch(ctx context.Context) {
	evch, errch := w.subscriber.Subscribe(ctx, w.processor.Filters()...)

	process := func(event events.Event) {
		err := w.processor.ProcessEvent(ctx, event)
		if err != nil {
			log.G(ctx).WithError(err).Errorf("watcher: failed to process event %T", event)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-evch:
			if !ok {
				return
			}

			if w.sync {
				process(event)
			} else {
				w.wg.Go(func() { process(event) })
			}

		case err := <-errch:
			if err != nil {
				log.G(ctx).WithError(err).Error("watcher: failed to read events from bus")
			}
		}
	}
}
