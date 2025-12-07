package events

import "fmt"

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
