package engine

import (
	"errors"

	"github.com/google/uuid"
	"github.com/sazonovItas/mini-ci/workflower/model"
)

const (
	initPlanName    = "init"
	cleanupPlanName = "clean up"
)

var (
	ErrEmptyJobSet    = errors.New("empty job set")
	ErrEmptyScriptSet = errors.New("empty script set")
)

type Planner struct{}

func NewPlanner() Planner {
	return Planner{}
}

func (p Planner) JobPlan(cfg model.WorkflowConfig) (model.JobPlan, error) {
	jobs := cfg.Jobs
	if len(jobs) == 0 {
		return model.JobPlan{}, ErrEmptyJobSet
	}

	var (
		headPlan = &model.JobPlan{}
		currPlan = headPlan
	)
	for _, job := range jobs {
		currPlan.Next = &model.JobPlan{
			Ref:    &model.OriginRef{ID: p.nextID()},
			Config: job,
		}

		currPlan = currPlan.Next
	}

	return *headPlan.Next, nil
}

func (p Planner) TaskPlan(cfg model.JobConfig) (model.TaskPlan, error) {
	if len(cfg.Run.Scripts) == 0 {
		return model.TaskPlan{}, nil
	}

	headPlan := &model.TaskPlan{
		Ref: &model.OriginRef{ID: p.nextID()},
		Config: model.Step{
			Config: &model.InitStep{
				Name:  initPlanName,
				Image: cfg.Run.Image,
				Cwd:   cfg.Run.Cwd,
				Env:   cfg.Run.Env,
			},
		},
	}

	currPlan := headPlan
	for _, script := range cfg.Run.Scripts {
		currPlan.Next = &model.TaskPlan{
			Ref: &model.OriginRef{ID: p.nextID()},
			Config: model.Step{
				Config: &model.ScriptStep{
					Name:    script.Name,
					Command: script.Command,
					Args:    script.Args,
				},
			},
		}

		currPlan = currPlan.Next
	}

	currPlan.Next = &model.TaskPlan{
		Ref: &model.OriginRef{ID: p.nextID()},
		Config: model.Step{
			Config: &model.CleanupStep{
				Name: cleanupPlanName,
			},
		},
	}

	return *headPlan, nil
}

func (p Planner) nextID() string {
	return uuid.New().String()
}
