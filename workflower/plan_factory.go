package workflower

import (
	"fmt"

	"github.com/google/uuid"
)

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
