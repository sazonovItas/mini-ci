package events

import (
	"encoding/json"
	"fmt"
)

type Message struct {
	Event Event
}

type Envelope struct {
	Event   EventType        `json:"event"`
	Payload *json.RawMessage `json:"payload"`
}

func (m Message) MarshalJSON() ([]byte, error) {
	var envelope Envelope

	payload, err := json.Marshal(m.Event)
	if err != nil {
		return nil, err
	}

	envelope.Event = m.Event.Type()
	envelope.Payload = (*json.RawMessage)(&payload)

	return json.Marshal(envelope)
}

func (m *Message) UnmarshalJSON(bytes []byte) error {
	var envelope Envelope

	err := json.Unmarshal(bytes, &envelope)
	if err != nil {
		return err
	}

	event, err := ParseEvent(envelope.Event, *envelope.Payload)
	if err != nil {
		return err
	}

	m.Event = event

	return nil
}

type UnknownEventError struct {
	Type EventType
}

func (err UnknownEventError) Error() string {
	return fmt.Sprintf("unkown event type: %s", err.Type)
}

func ParseEvent(eventType EventType, payload json.RawMessage) (Event, error) {
	parser, found := events[eventType]
	if !found {
		return nil, UnknownEventError{Type: eventType}
	}

	event, err := parser(payload)
	if err != nil {
		return nil, err
	}

	return event, nil
}
