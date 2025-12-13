package model

import "github.com/google/uuid"

type Plan struct {
	Ref  *OriginRef `json:"ref"`
	Next *Plan      `json:"next"`
}

type OriginRef struct {
	ID string `json:"id"`
}

type JobPlan struct {
	Plan   `json:",inline"`
	Config JobConfig `json:"config"`
}

type TaskPlan struct {
	Plan   `json:",inline"`
	Config Step `json:"config"`
}

type PlanFactory struct{}

func (f PlanFactory) JobPlan(cfg WorkflowConfig) JobPlan {
	return JobPlan{}
}

func (f PlanFactory) TaskPlan(cfg JobConfig) TaskPlan {
	return TaskPlan{}
}

func (f PlanFactory) nextID() string {
	return uuid.New().String()
}
