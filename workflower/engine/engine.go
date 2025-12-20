package engine

import (
	"context"
	"errors"

	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/core/events/watcher"
	"github.com/sazonovItas/mini-ci/workflower/db"
)

var (
	ErrBuildNotFound = errors.New("build not found")
	ErrJobNotFound   = errors.New("job not found")
	ErrTaskNotFound  = errors.New("task not found")
)

type Engine struct {
	bus events.Bus

	builds *db.BuildFactory
	jobs   *db.JobFactory
	tasks  *db.TaskFactory

	watchers []*watcher.Watcher
}

func New(
	bus events.Bus,
	builds *db.BuildFactory,
	jobs *db.JobFactory,
	tasks *db.TaskFactory,
) *Engine {
	return &Engine{
		bus: bus,

		builds: builds,
		jobs:   jobs,
		tasks:  tasks,

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
