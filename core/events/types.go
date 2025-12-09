package events

type EventType string

const (
	EventTypeStartInitContainer  EventType = "start-init-container"
	EventTypeFinishInitContainer EventType = "finish-init-container"

	EventTypeStartScript  EventType = "start-script"
	EventTypeFinishScript EventType = "finish-script"

	EventTypeLog EventType = "log"

	EventTypeError EventType = "error"
)
