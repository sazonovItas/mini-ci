package worker

import "github.com/sazonovItas/mini-ci/core/events"

type eventLoggerFactory struct {
	sender Sender
}

func NewEventLoggerFactory(sender Sender) *eventLoggerFactory {
	return &eventLoggerFactory{
		sender: sender,
	}
}

func (f *eventLoggerFactory) New(origin events.EventOrigin) EventLogger {
	return NewEventLogger(f.sender, origin)
}
