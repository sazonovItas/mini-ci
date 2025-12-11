package workflower

import "github.com/sazonovItas/mini-ci/core/status"

type Build struct {
	ID         string        `json:"id"`
	WorkflowID string        `json:"workflowId"`
	Status     status.Status `json:"status"`
	Plan       Plan          `json:"plan"`
}

type Job struct {
	ID      string        `json:"id"`
	BuildID string        `json:"buildId"`
	Status  status.Status `json:"status"`
	Plan    Plan          `json:"plan"`
}

type Task struct {
	ID     string        `json:"id"`
	JobID  string        `json:"jobId"`
	Name   string        `json:"name"`
	Status status.Status `json:"status"`
	Config Step          `json:"config"`
}

type Plan struct {
	Ref  *OriginRef `json:"ref"`
	Next *Plan      `json:"next"`
}

type OriginRef struct {
	ID string `json:"id"`
}
