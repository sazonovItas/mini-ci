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
	EventTypeScriptFinish EventType = "script:finish"

	EventTypeBuildStatus EventType = "build:status"
	EventTypeJobStatus   EventType = "job:status"
	EventTypeTaskStatus  EventType = "task:status"

	EventTypeBuildAbort EventType = "build:abort"
	EventTypeJobAbort   EventType = "job:abort"
	EventTypeTaskAbort  EventType = "task:abort"

	EventTypeTaskLog EventType = "task:log"

	EventTypeError EventType = "error"
)
