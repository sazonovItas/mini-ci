package model

type Workflow struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	CurrBuild *Build         `json:"currBuild"`
	Config    WorkflowConfig `json:"config"`
}

type WorkflowConfig struct {
	Jobs []JobConfig `json:"jobs"`
}

type JobConfig struct {
	Name string    `json:"name"`
	Run  RunConfig `json:"run"`
}

type RunConfig struct {
	Image   string         `json:"image"`
	Cwd     string         `json:"cwd,omitempty"`
	Env     []string       `json:"env,omitempty"`
	Scripts []ScriptConfig `json:"scripts"`
}

type ScriptConfig struct {
	Name    string   `json:"name"`
	Command []string `json:"command"`
	Args    []string `json:"args,omitempty"`
}
