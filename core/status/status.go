package status

type Status string

const (
	StatusCreated   Status = "created"
	StatusSkipped   Status = "skipped"
	StatusPending   Status = "pending"
	StatusStarted   Status = "started"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusErrored   Status = "errored"
	StatusAborted   Status = "aborted"
)

func (status Status) String() string {
	return string(status)
}
