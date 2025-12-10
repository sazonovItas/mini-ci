package workflower

import (
	"fmt"

	"github.com/google/uuid"
)

type PlanID string

func (id PlanID) String() string {
	return string(id)
}

type PlanFactory struct{}

type PlanConfig any

func (f PlanFactory) New(step PlanConfig) Plan {
	var plan Plan
	switch t := step.(type) {
	case JobPlan:
		plan.Job = &t
	case RunPlan:
		plan.Run = &t
	case ScriptPlan:
		plan.Script = &t
	case ContainerPlan:
		plan.Container = &t
	default:
		panic(fmt.Sprintf("cannot construct plan from %T", step))
	}

	plan.ID = f.nextID()

	return plan
}

func (f PlanFactory) nextID() PlanID {
	return PlanID(uuid.New().String())
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
