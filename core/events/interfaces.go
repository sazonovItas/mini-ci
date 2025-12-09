package events

import "context"

type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

type Subscriber interface {
	Subscribe(ctx context.Context, filters ...FilterFunc) (ch <-chan Event, errs <-chan error)
}
