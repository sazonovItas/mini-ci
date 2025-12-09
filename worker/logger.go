package worker

import (
	"time"

	"github.com/sazonovItas/mini-ci/core/events"
)

type eventLogger struct {
	sender Sender
	origin events.EventOrigin
}

func NewEventLogger(sender Sender, origin events.EventOrigin) *eventLogger {
	return &eventLogger{
		sender: sender,
		origin: origin,
	}
}

func (l *eventLogger) Log(msg string) {
	l.sendMessages([]events.LogMessage{l.newLogMessage(msg)})
}

func (l *eventLogger) Process(id string, stdout <-chan string, stderr <-chan string) error {
	const (
		batchSendTimeout = 1 * time.Second
	)

	ticker := time.NewTicker(batchSendTimeout)
	defer ticker.Stop()

	var (
		messages    []events.LogMessage
		isErrClosed = false
		isOutClosed = false
	)

	for !isOutClosed || !isErrClosed {
		select {
		case log, ok := <-stdout:
			if !ok {
				isOutClosed = true
				break
			}

			messages = append(messages, l.newLogMessage(log))

		case log, ok := <-stderr:
			if !ok {
				isErrClosed = true
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
	event := events.Log{
		EventOrigin: l.origin,
		Messages:    messages,
	}
	l.sender.Send(event)
}

func (l *eventLogger) newLogMessage(msg string) events.LogMessage {
	return events.LogMessage{Msg: msg, Time: time.Now().UTC()}
}
