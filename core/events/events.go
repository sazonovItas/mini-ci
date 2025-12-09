package events

import (
	"encoding/json"
	"time"
)

func init() {
	RegisterEvent[StartInitContainer]()
	RegisterEvent[FinishInitContainer]()
	RegisterEvent[StartScript]()
	RegisterEvent[FinishScript]()
	RegisterEvent[Error]()
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
	TaskID    string    `json:"task_id"`
	OccuredAt time.Time `json:"occured_at"`
}

func (o EventOrigin) Origin() EventOrigin {
	return o
}

type ContainerConfig struct {
	Image string   `json:"image"`
	Cwd   string   `json:"cwd,omitempty"`
	Env   []string `json:"env,omitempty"`
}

type StartInitContainer struct {
	EventOrigin `json:",inline"`
	Config      ContainerConfig `json:"config"`
}

func (StartInitContainer) Type() EventType { return EventTypeStartInitContainer }

type FinishInitContainer struct {
	EventOrigin `json:",inline"`
	ContainerID string `json:"containerId"`
}

func (FinishInitContainer) Type() EventType { return EventTypeFinishInitContainer }

type ScriptConfig struct {
	ContainerID string   `json:"containerId"`
	Command     []string `json:"command"`
	Args        []string `json:"args,omitempty"`
}

type StartScript struct {
	EventOrigin `json:",inline"`
	Config      ScriptConfig `json:"config"`
}

func (StartScript) Type() EventType { return EventTypeStartScript }

type FinishScript struct {
	EventOrigin `json:",inline"`
	ExitStatus  int  `json:"exitStatus"`
	Succeeded   bool `json:"succeeded"`
}

func (FinishScript) Type() EventType { return EventTypeFinishScript }

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
	return Error{
		EventOrigin: origin,
		Message:     msg,
	}
}
