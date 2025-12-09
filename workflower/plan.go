package workflower

type PlanID string

func (id PlanID) String() string {
	return string(id)
}

type Plan struct {
	ID PlanID `json:"id"`

	Job *JobPlan `json:"job,omitempty"`
	Run *RunPlan `json:"run,omitempty"`

	Script    *ScriptPlan    `json:"script,omitempty"`
	Container *ContainerPlan `json:"container,omitempty"`
}

func (p *Plan) Each(f func(*Plan)) {
	if p == nil {
		return
	}

	f(p)

	if p.Job != nil {
		p.Job.Plan.Each(f)
		p.Job.Next.Each(f)
	}

	if p.Run != nil {
		p.Run.Plan.Each(f)
		p.Run.Next.Each(f)
	}
}

type JobPlan struct {
	Plan *Plan `json:"plan"`
	Next *Plan `json:"next"`
}

type RunPlan struct {
	Plan *Plan `json:"plan"`
	Next *Plan `json:"next"`
}

type ContainerPlan struct {
	Image string   `json:"image"`
	Cwd   string   `json:"cwd,omitempty"`
	Env   []string `json:"env,omitempty"`
}

type ScriptPlan struct {
	Command []string `json:"command"`
	Args    []string `json:"args,omitempty"`
}
