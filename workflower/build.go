package workflower

import "time"

type BuildStatus string

const (
	BuildStatusStarted   BuildStatus = "started"
	BuildStatusPending   BuildStatus = "pending"
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
	Status     BuildStatus `json:"status"`
	Plan       Plan        `json:"plan"`
	StartedAt  *time.Time  `json:"startedAt,omitempty"`
	FinishedAt *time.Time  `json:"finishedAt,omitempty"`
}
