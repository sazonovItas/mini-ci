package events

type EventType string

const (
	EventTypeContainerInitStart  EventType = "container:init:start"
	EventTypeContainerInitFinish EventType = "container:init:finish"

	EventTypeScriptStart  EventType = "script:start"
	EventTypeScriptFinish EventType = "script:finish"

	EventTypeWorkerRegister EventType = "worker:register"

	EventTypeLog EventType = "log"

	EventTypeError EventType = "error"
)
