package workflower

type TaskStatus string

const (
	TaskStatusCreated   TaskStatus = "created"
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusStarted   TaskStatus = "started"
	TaskStatusSucceeded TaskStatus = "succeeded"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusErrored   TaskStatus = "errored"
	TaskStatusAborted   TaskStatus = "aborted"
)

func (status TaskStatus) String() string {
	return string(status)
}

type Task struct {
	ID      string     `json:"id"`
	BuildID string     `json:"buildId"`
	Status  TaskStatus `json:"status"`
	Step    Step       `json:"step"`
}
