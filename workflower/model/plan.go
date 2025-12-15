package model

type OriginRef struct {
	ID string `json:"id"`
}

type JobPlan struct {
	Ref    *OriginRef `json:"ref,omitempty"`
	Next   *JobPlan   `json:"next,omitempty"`
	Config JobConfig  `json:"config"`
}

type TaskPlan struct {
	Ref    *OriginRef `json:"ref,omitempty"`
	Next   *TaskPlan  `json:"next,omitempty"`
	Config Step       `json:"config"`
}
