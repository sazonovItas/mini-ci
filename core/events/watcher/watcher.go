package watcher

import (
	"context"
	"sync"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
)

type EventProcessor interface {
	Filters() []events.FilterFunc
	ProcessEvent(ctx context.Context, event events.Event) error
}

type Watcher struct {
	subscriber events.Subscriber
	processor  EventProcessor

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func NewWatcher(
	subscriber events.Subscriber,
	processor EventProcessor,
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

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-evch:
			if !ok {
				return
			}

			w.wg.Go(func() {
				err := w.processor.ProcessEvent(ctx, event)
				if err != nil {
					log.G(ctx).WithError(err).Errorf("watcher: failed to process event %T", event)
				}
			})

		case err := <-errch:
			if err != nil {
				log.G(ctx).WithError(err).Error("watcher: failed to read events from bus")
			}
		}
	}
}
