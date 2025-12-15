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

func (s Status) String() string {
	return string(s)
}

func (s Status) IsRunning() bool {
	return s == StatusPending || s == StatusStarted
}

func (s Status) IsFinished() bool {
	return s == StatusSkipped || s == StatusSucceeded ||
		s == StatusFailed || s == StatusErrored || s == StatusAborted
}

func (s Status) IsCreated() bool {
	return s == StatusCreated
}

func (s Status) IsPending() bool {
	return s == StatusPending
}

func (s Status) IsStarted() bool {
	return s == StatusStarted
}

func (s Status) IsSkipped() bool {
	return s == StatusSkipped
}

func (s Status) IsSucceeded() bool {
	return s == StatusSucceeded
}

func (s Status) IsFailed() bool {
	return s == StatusFailed
}

func (s Status) IsErrored() bool {
	return s == StatusErrored
}

func (s Status) IsAborted() bool {
	return s == StatusAborted
}
