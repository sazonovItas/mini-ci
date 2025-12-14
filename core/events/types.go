package events

type EventType string

func (t EventType) String() string {
	return string(t)
}

const (
	EventTypeContainerInitStart  EventType = "container:init:start"
	EventTypeContainerInitFinish EventType = "container:init:finish"
	EventTypeContainerDestroy    EventType = "container:destroy"

	EventTypeScriptStart  EventType = "script:start"
	EventTypeScriptFinish EventType = "script:finish"

	EventTypeBuildStatus EventType = "build:status"
	EventTypeJobStatus   EventType = "job:status"
	EventTypeTaskStatus  EventType = "task:status"

	EventTypeTaskLog EventType = "task:log"

	EventTypeError EventType = "error"
)
