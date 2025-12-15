package events

import (
	"context"
	"fmt"
	"time"
)

type ErrCannotConvertEvent struct {
	From EventType
	To   EventType
}

func (err ErrCannotConvertEvent) Error() string {
	return fmt.Sprintf("cannot convert event from %s to %s", err.From, err.To)
}

func ConvertTo[T Event](e Event) (T, error) {
	var value T
	if e.Type() != value.Type() {
		return value, ErrCannotConvertEvent{From: e.Type(), To: value.Type()}
	}

	return e.(T), nil
}

func PublishAfter(ctx context.Context, publisher Publisher, event Event, timeout time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(timeout):
		_ = publisher.Publish(ctx, event)
	}
}
