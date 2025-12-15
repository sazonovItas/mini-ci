package events

type EventType string

func (t EventType) String() string {
	return string(t)
}

const (
	EventTypeInitContainerStart  EventType = "init:container:start"
	EventTypeInitContainerFinish EventType = "init:container:finish"
	EventTypeCleanupContainer    EventType = "cleanup:container"

	EventTypeScriptStart  EventType = "script:start"
	EventTypeScriptAbort  EventType = "script:abort"
	EventTypeScriptFinish EventType = "script:finish"

	EventTypeBuildStatus EventType = "build:status"
	EventTypeJobStatus   EventType = "job:status"
	EventTypeTaskStatus  EventType = "task:status"

	EventTypeTaskLog EventType = "task:log"

	EventTypeError EventType = "error"
)
