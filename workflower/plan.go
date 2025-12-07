package workflower

type Plan struct {
	Step *StepPlan `json:"step,omitempty"`
	Run  *RunPlan  `json:"run,omitempty"`

	Script    *ScriptPlan    `json:"script,omitempty"`
	Container *ContainerPlan `json:"container,omitempty"`
}

func (p *Plan) Each(f func(*Plan)) {
	if p == nil {
		return
	}

	f(p)

	if p.Step != nil {
		p.Step.Plan.Each(f)
		p.Step.Next.Each(f)
	}

	if p.Run != nil {
		p.Run.Plan.Each(f)
		p.Run.Next.Each(f)
	}
}

type StepPlan struct {
	Plan *Plan `json:"plan"`
	Next *Plan `json:"next"`
}

type RunPlan struct {
	Plan *Plan `json:"plan"`
	Next *Plan `json:"next"`
}

type ContainerPlan struct {
	Image string `json:"image"`
	Env   string `json:"env,omitempty"`
	Cwd   string `json:"cwd,omitempty"`
}

type ScriptPlan struct {
	Command []string `json:"command"`
	Args    []string `json:"args,omitempty"`
}
