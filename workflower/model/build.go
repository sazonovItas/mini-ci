package model

import "github.com/sazonovItas/mini-ci/core/status"

type Build struct {
	ID         string         `json:"id"`
	WorkflowID string         `json:"workflowId"`
	Status     status.Status  `json:"status"`
	Config     WorkflowConfig `json:"config"`
	Plan       JobPlan        `json:"plan"`
}

type Job struct {
	ID      string        `json:"id"`
	BuildID string        `json:"buildId"`
	Name    string        `json:"name"`
	Status  status.Status `json:"status"`
	Config  JobConfig     `json:"config"`
	Plan    TaskPlan      `json:"plan"`
}

type Task struct {
	ID     string        `json:"id"`
	JobID  string        `json:"jobId"`
	Name   string        `json:"name"`
	Status status.Status `json:"status"`
	Config Step          `json:"config"`
}
