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
}

type OriginID string

func (id OriginID) String() string {
	return string(id)
}

type Origin struct {
	ID OriginID `json:"id,omitempty"`
}

type ContainerConfig struct {
	Image string   `json:"image"`
	Cwd   string   `json:"cwd,omitempty"`
	Env   []string `json:"env,omitempty"`
}

type StartInitContainer struct {
	Origin Origin          `json:"origin"`
	Time   time.Time       `json:"time"`
	Config ContainerConfig `json:"config"`
}

func (StartInitContainer) Type() EventType { return EventTypeStartInitContainer }

type FinishInitContainer struct {
	Origin      Origin    `json:"origin"`
	Time        time.Time `json:"time"`
	ContainerID string    `json:"containerId"`
	Succeeded   bool      `json:"succeeded"`
}

func (FinishInitContainer) Type() EventType { return EventTypeFinishInitContainer }

type ScriptConfig struct {
	ContainerID string   `json:"containerId"`
	Command     []string `json:"command"`
	Args        []string `json:"args,omitempty"`
}

type StartScript struct {
	Origin Origin       `json:"origin"`
	Time   time.Time    `json:"time"`
	Config ScriptConfig `json:"config"`
}

func (StartScript) Type() EventType { return EventTypeStartScript }

type FinishScript struct {
	Origin     Origin    `json:"origin"`
	Time       time.Time `json:"time"`
	ExitStatus int       `json:"exitStatus"`
	Succeeded  bool      `json:"succeeded"`
}

func (FinishScript) Type() EventType { return EventTypeFinishScript }

type Error struct {
	Origin  Origin    `json:"origin"`
	Time    time.Time `json:"time"`
	Message string    `json:"message"`
}

func (Error) Type() EventType { return EventTypeError }
