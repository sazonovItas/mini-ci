package engine

import (
	"github.com/sazonovItas/mini-ci/workflower/db"
)

type Engine struct {
	builds *db.BuildFactory
	jobs   *db.JobFactory
	tasks  *db.TaskFactory

	planner Planner
}

func New(builds *db.BuildFactory, jobs *db.JobFactory, tasks *db.TaskFactory) *Engine {
	return &Engine{
		builds: builds,
		jobs:   jobs,
		tasks:  tasks,

		planner: NewPlanner(),
	}
}
