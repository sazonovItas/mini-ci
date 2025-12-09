package workflower

import "time"

type BuildStatus string

const (
	BuildStatusCreated   BuildStatus = "created"
	BuildStatusPending   BuildStatus = "pending"
	BuildStatusStarted   BuildStatus = "started"
	BuildStatusSucceeded BuildStatus = "succeeded"
	BuildStatusFailed    BuildStatus = "failed"
	BuildStatusErrored   BuildStatus = "errored"
	BuildStatusAborted   BuildStatus = "aborted"
)

func (status BuildStatus) String() string {
	return string(status)
}

type Build struct {
	ID         string      `json:"id"`
	WorkflowID string      `json:"workflowId"`
	Status     BuildStatus `json:"status"`
	Plan       Plan        `json:"plan"`
	StartedAt  *time.Time  `json:"startedAt,omitempty"`
	FinishedAt *time.Time  `json:"finishedAt,omitempty"`
}

func (b Build) IsRunning() bool {
	if b.Status == BuildStatusPending || b.Status == BuildStatusStarted {
		return true
	}

	return false
}
