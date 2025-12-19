package worker

import (
	"context"
	"time"

	"github.com/sazonovItas/mini-ci/core/events"
)

type eventLoggerFactory struct {
	publisher events.Publisher
}

func NewEventLoggerFactory(publisher events.Publisher) *eventLoggerFactory {
	return &eventLoggerFactory{
		publisher: publisher,
	}
}

func (f *eventLoggerFactory) New(origin events.EventOrigin) EventLogger {
	return NewEventLogger(f.publisher, origin)
}

type eventLogger struct {
	publisher events.Publisher
	origin    events.EventOrigin
}

func NewEventLogger(publisher events.Publisher, origin events.EventOrigin) *eventLogger {
	return &eventLogger{
		publisher: publisher,
		origin:    origin,
	}
}

func (l *eventLogger) Log(msg string) {
	l.sendMessages([]events.LogMessage{l.newLogMessage(msg)})
}

func (l *eventLogger) Process(id string, stdout <-chan string, stderr <-chan string) error {
	const (
		messagesSendTimeout = 5 * time.Second
	)

	ticker := time.NewTicker(messagesSendTimeout)
	defer ticker.Stop()

	var (
		messages    []events.LogMessage
		isErrClosed = false
		isOutClosed = false
	)

	defer func() {
		l.sendMessages(messages)
	}()

	for !isOutClosed || !isErrClosed {
		select {
		case log, ok := <-stdout:
			if !ok {
				isOutClosed = true
				stdout = nil
				break
			}

			messages = append(messages, l.newLogMessage(log))

		case log, ok := <-stderr:
			if !ok {
				isErrClosed = true
				stderr = nil
				break
			}

			messages = append(messages, l.newLogMessage(log))

		case <-ticker.C:
			l.sendMessages(messages)
			messages = messages[:0]
		}
	}

	return nil
}

func (l *eventLogger) sendMessages(messages []events.LogMessage) {
	if len(messages) == 0 {
		return
	}

	_ = l.publisher.Publish(
		context.TODO(),
		events.TaskLog{EventOrigin: l.origin, Messages: messages},
	)
}

func (l *eventLogger) newLogMessage(msg string) events.LogMessage {
	return events.LogMessage{Msg: msg, Time: time.Now().UTC()}
}
