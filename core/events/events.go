package events

import (
	"encoding/json"
	"time"

	"github.com/sazonovItas/mini-ci/core/status"
)

func init() {
	registerEvent[InitContainerStart]()
	registerEvent[InitContainerFinish]()
	registerEvent[CleanupContainer]()
	registerEvent[ScriptStart]()
	registerEvent[ScriptFinish]()
	registerEvent[TaskAbort]()
	registerEvent[Log]()
	registerEvent[Error]()
	registerEvent[BuildStatus]()
	registerEvent[JobStatus]()
	registerEvent[TaskStatus]()
}

func registerEvent[T Event]() {
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
	ID        string    `json:"id"`
	OccuredAt time.Time `json:"occuredAt"`
}

func NewEventOrigin(id string) EventOrigin {
	return EventOrigin{ID: id, OccuredAt: time.Now().UTC()}
}

func (o EventOrigin) Origin() EventOrigin {
	return o
}

type ContainerConfig struct {
	Image string   `json:"image"`
	Cwd   string   `json:"cwd,omitempty"`
	Env   []string `json:"env,omitempty"`
}

type InitContainerStart struct {
	EventOrigin `json:",inline"`
	Config      ContainerConfig `json:"config"`
}

func (InitContainerStart) Type() EventType { return EventTypeInitContainerStart }

type InitContainerFinish struct {
	EventOrigin `json:",inline"`
	ContainerID string `json:"containerId"`
}

func (InitContainerFinish) Type() EventType { return EventTypeInitContainerFinish }

type CleanupContainer struct {
	EventOrigin `json:",inline"`
	ContainerID string `json:"containerId"`
}

func (CleanupContainer) Type() EventType { return EventTypeCleanupContainer }

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

type BuildAbort struct {
	EventOrigin `json:",inline"`
}

func (BuildAbort) Type() EventType { return EventTypeBuildAbort }

type JobAbort struct {
	EventOrigin `json:",inline"`
}

func (JobAbort) Type() EventType { return EventTypeJobAbort }

type TaskAbort struct {
	EventOrigin `json:",inline"`
	ContainerID string `json:"containerId"`
}

func (TaskAbort) Type() EventType { return EventTypeTaskAbort }

type LogMessage struct {
	Msg  string    `json:"msg"`
	Time time.Time `json:"time"`
}

type Log struct {
	EventOrigin `json:",inline"`
	Messages    []LogMessage `json:"messages"`
}

func (Log) Type() EventType { return EventTypeTaskLog }

type Error struct {
	EventOrigin `json:",inline"`
	Message     string `json:"message"`
}

func (Error) Type() EventType { return EventTypeError }

func NewErrorEvent(origin EventOrigin, msg string) Error {
	return Error{EventOrigin: origin, Message: msg}
}

type ChangeStatus struct {
	EventOrigin `json:",inline"`
	Status      status.Status `json:"status"`
}

type BuildStatus ChangeStatus

func (BuildStatus) Type() EventType { return EventTypeBuildStatus }

type JobStatus struct {
	ChangeStatus `json:",inline"`
	BuildID      string `json:"buildId"`
}

func (JobStatus) Type() EventType { return EventTypeJobStatus }

type TaskStatus struct {
	ChangeStatus `json:",inline"`
	JobID        string `json:"jobId"`
}

func (TaskStatus) Type() EventType { return EventTypeTaskStatus }
