package exchange

import (
	"context"
	"errors"
	"fmt"

	"github.com/containerd/log"
	goevents "github.com/docker/go-events"
	"github.com/sazonovItas/mini-ci/core/events"
)

type Exchange struct {
	broadcaster *goevents.Broadcaster
}

func NewExchange() *Exchange {
	return &Exchange{
		broadcaster: goevents.NewBroadcaster(),
	}
}

var (
	_ events.Publisher  = (*Exchange)(nil)
	_ events.Subscriber = (*Exchange)(nil)
)

func (e *Exchange) Publish(ctx context.Context, event events.Event) (err error) {
	defer func() {
		logger := log.G(ctx).WithFields(log.Fields{
			"event": event.Type(),
		})

		if err != nil {
			logger.WithError(err).Error("failed to publish event")
		} else {
			logger.Trace("event published")
		}
	}()

	return e.broadcaster.Write(event)
}

func (e *Exchange) Subscribe(ctx context.Context, filters ...events.FilterFunc) (ch <-chan events.Event, errs <-chan error) {
	var (
		evch    = make(chan events.Event)
		errq    = make(chan error, 1)
		channel = goevents.NewChannel(0)
		queue   = goevents.NewQueue(channel)
		dst     = queue
	)

	closeAll := func() {
		_ = channel.Close()
		_ = queue.Close()
		_ = e.broadcaster.Remove(dst)
		close(errq)
	}

	ch = evch
	errs = errq

	_ = e.broadcaster.Add(dst)

	go func() {
		defer closeAll()

		var err error
	loop:
		for {
			select {
			case event := <-channel.C:
				ev, ok := event.(events.Event)
				if !ok {
					err = fmt.Errorf("invalid event %#v", ev)
					break
				}

				if len(filters) > 0 {
					var filtered bool
					for _, filter := range filters {
						if !filter(ev) {
							filtered = true
							break
						}
					}

					if filtered {
						break
					}
				}

				select {
				case evch <- ev:
				case <-ctx.Done():
					break loop
				}
			case <-ctx.Done():
				break loop
			}

			if err == nil {
				if cerr := ctx.Err(); !errors.Is(cerr, context.Canceled) {
					err = cerr
				}
			}

			errq <- err
		}
	}()

	return
}
