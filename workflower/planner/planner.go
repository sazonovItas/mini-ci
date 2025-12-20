package planner

import (
	"github.com/google/uuid"
	"github.com/sazonovItas/mini-ci/core/status"
	"github.com/sazonovItas/mini-ci/workflower/model"
)

type PlanOutput struct {
	Build model.Build
	Jobs  []model.Job
	Tasks []model.Task
}

type Planner struct {
	factory PlanFactory
}

func NewPlanner() Planner {
	return Planner{
		factory: NewPlanFactory(),
	}
}

func (p Planner) Plan(workflow model.Workflow) (PlanOutput, error) {
	build, err := p.build(workflow)
	if err != nil {
		return PlanOutput{}, err
	}

	var (
		jobs        []model.Job
		currJobPlan = &build.Plan
	)
	for currJobPlan != nil {
		job, err := p.job(currJobPlan.Ref.ID, build.ID, currJobPlan.Config)
		if err != nil {
			return PlanOutput{}, err
		}

		jobs = append(jobs, job)

		currJobPlan = currJobPlan.Next
	}

	var tasks []model.Task
	for _, job := range jobs {
		currTaskPlan := &job.Plan

		for currTaskPlan != nil {
			tasks = append(
				tasks,
				p.task(
					currTaskPlan.Ref.ID,
					job.ID,
					currTaskPlan.Config,
				),
			)

			currTaskPlan = currTaskPlan.Next
		}
	}

	output := PlanOutput{
		Build: build,
		Jobs:  jobs,
		Tasks: tasks,
	}

	return output, nil
}

func (p Planner) build(workflow model.Workflow) (model.Build, error) {
	jobPlan, err := p.factory.JobPlan(workflow.Config)
	if err != nil {
		return model.Build{}, err
	}

	build := model.Build{
		ID:         p.nextID(),
		WorkflowID: workflow.ID,
		Status:     status.StatusCreated,
		Config:     workflow.Config,
		Plan:       jobPlan,
	}

	return build, nil
}

func (p Planner) job(jobID, buildID string, cfg model.JobConfig) (model.Job, error) {
	taskPlan, err := p.factory.TaskPlan(cfg)
	if err != nil {
		return model.Job{}, err
	}

	job := model.Job{
		ID:      jobID,
		BuildID: buildID,
		Name:    cfg.Name,
		Status:  status.StatusCreated,
		Config:  cfg,
		Plan:    taskPlan,
	}

	return job, nil
}

func (p Planner) task(taskID, jobID string, cfg model.Step) model.Task {
	task := model.Task{
		ID:     taskID,
		JobID:  jobID,
		Name:   cfg.Config.StepName(),
		Status: status.StatusCreated,
		Config: cfg,
	}

	return task
}

func (p Planner) nextID() string {
	return uuid.New().String()
}
