package engine

import (
	"context"

	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/core/events/watcher"
	"github.com/sazonovItas/mini-ci/workflower/db"
)

type Engine struct {
	bus events.Bus

	workflows *db.WorkflowFactory
	builds    *db.BuildFactory
	jobs      *db.JobFactory
	tasks     *db.TaskFactory

	planner  Planner
	watchers []*watcher.Watcher
}

func New(
	bus events.Bus,
	workflows *db.WorkflowFactory,
	builds *db.BuildFactory,
	jobs *db.JobFactory,
	tasks *db.TaskFactory,
) *Engine {
	return &Engine{
		bus: bus,

		workflows: workflows,
		builds:    builds,
		jobs:      jobs,
		tasks:     tasks,

		planner: NewPlanner(),
		watchers: []*watcher.Watcher{
			watcher.NewWatcher(bus, NewBuildProcessor(builds, jobs, bus)),
			watcher.NewWatcher(bus, NewJobProcessor(jobs, tasks, bus)),
			watcher.NewWatcher(bus, NewTaskProcessor(tasks, bus)),
		},
	}
}

func (e *Engine) Start(ctx context.Context) {
	for _, w := range e.watchers {
		w.Start(ctx)
	}
}

func (e *Engine) Stop() {
	for _, w := range e.watchers {
		w.Stop()
	}
}
