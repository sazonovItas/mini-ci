package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/sazonovItas/mini-ci/core/status"
	"github.com/sazonovItas/mini-ci/workflower/db/gen/psql"
	"github.com/sazonovItas/mini-ci/workflower/model"
)

type TaskFactory struct {
	queries *Queries
}

func NewTaskFactory(queries *Queries) *TaskFactory {
	return &TaskFactory{queries: queries}
}

func (f *TaskFactory) New(ctx context.Context, t model.Task) (Task, error) {
	queries := f.queries.Queries(ctx)

	config, err := json.Marshal(t.Config)
	if err != nil {
		return nil, err
	}

	got, err := queries.CreateTask(
		ctx,
		psql.CreateTaskParams{
			ID:     t.ID,
			JobID:  t.JobID,
			Name:   t.Name,
			Status: status.StatusCreated.String(),
			Config: config,
		},
	)
	if err != nil {
		return nil, err
	}

	return newTask(got, f.queries), nil
}

func (f *TaskFactory) Task(ctx context.Context, id string) (Task, bool, error) {
	queries := f.queries.Queries(ctx)

	got, err := queries.Task(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return newTask(got, f.queries), true, nil
}

func (f *TaskFactory) TasksByStatus(ctx context.Context, status status.Status) ([]Task, error) {
	queries := f.queries.Queries(ctx)

	tasks, err := queries.TasksByStatus(ctx, status.String())
	if err != nil {
		return nil, err
	}

	dbTasks := make([]Task, 0, len(tasks))
	for _, t := range tasks {
		dbTasks = append(dbTasks, newTask(t, f.queries))
	}

	return dbTasks, nil
}

func (f *TaskFactory) WithTx(ctx context.Context, txFunc func(txCtx context.Context) error) error {
	return f.queries.WithTx(ctx, txFunc)
}

func newTask(t psql.Task, q *Queries) *task {
	return &task{Task: t, queries: q}
}

type Task interface {
	Transactor

	ID() string
	JobID() string
	Name() string
	Status() status.Status
	Config() model.Step
	Model() model.Task

	Lock(ctx context.Context) error

	Pending(ctx context.Context) error
	Start(ctx context.Context) error
	Abort(ctx context.Context) error
	Finish(ctx context.Context, status status.Status) error

	UpdateConfig(ctx context.Context, config model.Step) error
}

type task struct {
	psql.Task

	queries *Queries
}

func (t *task) ID() string {
	return t.Task.ID
}

func (t *task) JobID() string {
	return t.Task.JobID
}

func (t *task) Name() string {
	return t.Task.Name
}

func (t *task) Status() status.Status {
	return status.Status(t.Task.Status)
}

func (t *task) Config() model.Step {
	var config model.Step
	_ = json.Unmarshal(t.Task.Config, &config)
	return config
}

func (t *task) Model() model.Task {
	return model.Task{
		ID:     t.ID(),
		JobID:  t.JobID(),
		Name:   t.Name(),
		Status: t.Status(),
		Config: t.Config(),
	}
}

func (t *task) Lock(ctx context.Context) error {
	queries := t.queries.Queries(ctx)

	lockedTask, err := queries.LockTask(ctx, t.ID())
	if err != nil {
		return err
	}

	t.Task = lockedTask

	return nil
}

func (t *task) Pending(ctx context.Context) error {
	queries := t.queries.Queries(ctx)

	if t.Status().IsRunning() {
		return ErrAlreadyRunning
	}

	if t.Status().IsFinished() {
		return ErrAlreadyFinished
	}

	updatedTask, err := queries.UpdateTaskStatus(
		ctx,
		psql.UpdateTaskStatusParams{
			ID:     t.ID(),
			Status: status.StatusPending.String(),
		},
	)
	if err != nil {
		return err
	}

	t.Task = updatedTask

	return nil
}

func (t *task) Start(ctx context.Context) error {
	queries := t.queries.Queries(ctx)

	if t.Status().IsFinished() {
		return ErrAlreadyFinished
	}

	updatedTask, err := queries.UpdateTaskStatus(
		ctx,
		psql.UpdateTaskStatusParams{
			ID:     t.ID(),
			Status: status.StatusStarted.String(),
		},
	)
	if err != nil {
		return err
	}
	t.Task = updatedTask

	return nil
}

func (t *task) Finish(ctx context.Context, status status.Status) error {
	queries := t.queries.Queries(ctx)

	if t.Status().IsFinished() {
		return ErrAlreadyFinished
	}

	updatedTask, err := queries.UpdateTaskStatus(
		ctx,
		psql.UpdateTaskStatusParams{
			ID:     t.ID(),
			Status: status.String(),
		},
	)
	if err != nil {
		return err
	}

	t.Task = updatedTask

	return nil
}

func (t *task) Abort(ctx context.Context) error {
	return t.Finish(ctx, status.StatusAborted)
}

func (t *task) UpdateConfig(ctx context.Context, cfg model.Step) error {
	queries := t.queries.Queries(ctx)

	config, err := json.Marshal(cfg)
	if err != nil {
		return nil
	}

	updatedTask, err := queries.UpdateTaskConfig(
		ctx,
		psql.UpdateTaskConfigParams{
			ID:     t.ID(),
			Config: config,
		},
	)
	if err != nil {
		return err
	}

	t.Task = updatedTask

	return nil
}

func (t *task) WithTx(ctx context.Context, txFunc func(txCtx context.Context) error) error {
	return t.queries.WithTx(ctx, txFunc)
}
