package workflower

type Workflow struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Config WorkflowConfig `json:"config"`
}

type WorkflowConfig struct {
	Jobs []JobConfig `json:"jobs"`
}

type JobConfig struct {
	Run RunConfig `json:"run"`
}

type RunConfig struct {
	Image   string         `json:"image"`
	Cwd     string         `json:"cwd,omitempty"`
	Env     []string       `json:"env,omitempty"`
	Scripts []ScriptConfig `json:"scripts"`
}

type ScriptConfig struct {
	Command []string `json:"command"`
	Args    []string `json:"args,omitempty"`
}
