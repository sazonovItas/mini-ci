package events

import (
	"encoding/json"
	"time"
)

func init() {
	RegisterEvent[Log]()
	RegisterEvent[Error]()
	RegisterEvent[WorkerRegister]()
	RegisterEvent[ContainerInitStart]()
	RegisterEvent[ContainerInitFinish]()
	RegisterEvent[ScriptStart]()
	RegisterEvent[ScriptFinish]()
}

func RegisterEvent[T Event]() {
	var e T
	if _, found := events[e.Type()]; !found {
		events[e.Type()] = unmarshaler[T]()
	}
}

type (
	eventTable  map[EventType]eventParser
	eventParser func([]byte) (Event, error)
)

var events = eventTable{}

func unmarshaler[T Event]() func([]byte) (Event, error) {
	return func(payload []byte) (Event, error) {
		var event T
		err := json.Unmarshal(payload, &event)
		return event, err
	}
}

type Event interface {
	Type() EventType
	Origin() EventOrigin
}

type EventOrigin struct {
	TaskID    string    `json:"taskId"`
	OccuredAt time.Time `json:"occuredAt"`
}

func NewEventOrigin(taskID string) EventOrigin {
	return EventOrigin{TaskID: taskID, OccuredAt: time.Now().UTC()}
}

func (o EventOrigin) Origin() EventOrigin {
	return o
}

type WorkerRegister struct {
	Name string `json:"name"`
}

func (WorkerRegister) Origin() EventOrigin { return EventOrigin{} }

func (WorkerRegister) Type() EventType { return EventTypeWorkerRegister }

type ContainerConfig struct {
	Image string   `json:"image"`
	Cwd   string   `json:"cwd,omitempty"`
	Env   []string `json:"env,omitempty"`
}

type ContainerInitStart struct {
	EventOrigin `json:",inline"`
	Config      ContainerConfig `json:"config"`
}

func (ContainerInitStart) Type() EventType { return EventTypeContainerInitStart }

type ContainerInitFinish struct {
	EventOrigin `json:",inline"`
	ContainerID string `json:"containerId"`
}

func (ContainerInitFinish) Type() EventType { return EventTypeContainerInitFinish }

type ScriptConfig struct {
	ContainerID string   `json:"containerId"`
	Command     []string `json:"command"`
	Args        []string `json:"args,omitempty"`
}

type ScriptStart struct {
	EventOrigin `json:",inline"`
	Config      ScriptConfig `json:"config"`
}

func (ScriptStart) Type() EventType { return EventTypeScriptStart }

type ScriptFinish struct {
	EventOrigin `json:",inline"`
	ExitStatus  int  `json:"exitStatus"`
	Succeeded   bool `json:"succeeded"`
}

func (ScriptFinish) Type() EventType { return EventTypeScriptFinish }

type LogMessage struct {
	Msg  string    `json:"msg"`
	Time time.Time `json:"time"`
}

type Log struct {
	EventOrigin `json:",inline"`
	Messages    []LogMessage `json:"messages"`
}

func (Log) Type() EventType { return EventTypeLog }

type Error struct {
	EventOrigin `json:",inline"`
	Message     string `json:"message"`
}

func (Error) Type() EventType { return EventTypeError }

func NewErrorEvent(origin EventOrigin, msg string) Error {
	return Error{EventOrigin: origin, Message: msg}
}
