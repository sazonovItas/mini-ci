package workflower

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type Step struct {
	Config        StepConfig
	UnknownFields map[string]*json.RawMessage
}

var (
	ErrNoStepDetected   = errors.New("no step detected")
	ErrNoStepConfigured = errors.New("no configured step")
)

func (s *Step) UnmarshalJSON(data []byte) error {
	var rawStepConfig map[string]*json.RawMessage
	err := json.Unmarshal(data, &rawStepConfig)
	if err != nil {
		return err
	}

	var stepDetected bool
	for _, detector := range StepDetectors {
		_, found := rawStepConfig[detector.Key]
		if !found {
			continue
		}

		step := detector.New()

		err := json.Unmarshal(data, step)
		if err != nil {
			return err
		}

		deleteKnownFields(rawStepConfig, step)

		data, err = json.Marshal(rawStepConfig)
		if err != nil {
			return fmt.Errorf("re-marshal rawStepConfig: %w", err)
		}
	}

	if s.Config == nil {
		return ErrNoStepConfigured
	}

	if !stepDetected {
		return ErrNoStepDetected
	}

	if len(rawStepConfig) != 0 {
		s.UnknownFields = rawStepConfig
	}

	return nil
}

func (s Step) MarshalJSON() ([]byte, error) {
	config := s.Config

	payload, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func deleteKnownFields(rawStepConfig map[string]*json.RawMessage, step StepConfig) {
	stepType := reflect.TypeOf(step).Elem()
	for i := 0; i < stepType.NumField(); i++ {
		field := stepType.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		jsonTagParts := strings.Split(jsonTag, ",")
		if len(jsonTagParts) < 1 {
			continue
		}
		jsonKey := jsonTagParts[0]
		if jsonKey == "" {
			jsonKey = field.Name
		}
		delete(rawStepConfig, jsonKey)
	}
}

type StepDetector struct {
	Key string
	New func() StepConfig
}

var StepDetectors = []StepDetector{
	{
		Key: "container",
		New: func() StepConfig { return &ContainerInitStep{} },
	},
	{
		Key: "script",
		New: func() StepConfig { return &ScriptStep{} },
	},
}

type StepConfig any

type ContainerOutputs struct {
	ContainerID string `json:"containerId"`
}

type ContainerInitStep struct {
	Name    string            `json:"container"`
	Image   []string          `json:"image"`
	Env     []string          `json:"env,omitempty"`
	Outputs *ContainerOutputs `json:"outputs,omitempty"`
}

type ScriptOutputs struct {
	ExitStatus int  `json:"exit_status"`
	Success    bool `json:"success"`
}

type ScriptStep struct {
	Name        string         `json:"script"`
	ContainerID string         `json:"containerId"`
	Command     []string       `json:"command"`
	Args        []string       `json:"args,omitempty"`
	Outputs     *ScriptOutputs `json:"outputs,omitempty"`
}
