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

type JobFactory struct {
	queries *Queries
}

func NewJobFactory(queries *Queries) *JobFactory {
	return &JobFactory{queries: queries}
}

func (f *JobFactory) New(ctx context.Context, j model.Job) (Job, error) {
	queries := f.queries.Queries(ctx)

	config, err := json.Marshal(j.Config)
	if err != nil {
		return nil, err
	}

	plan, err := json.Marshal(j.Plan)
	if err != nil {
		return nil, err
	}

	got, err := queries.CreateJob(
		ctx,
		psql.CreateJobParams{
			ID:      j.ID,
			BuildID: j.BuildID,
			Name:    j.Name,
			Status:  status.StatusCreated.String(),
			Config:  config,
			Plan:    plan,
		})
	if err != nil {
		return nil, err
	}

	return newJob(got, f.queries), nil
}

func (f *JobFactory) Job(ctx context.Context, id string) (Job, bool, error) {
	queries := f.queries.Queries(ctx)

	got, err := queries.Job(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return newJob(got, f.queries), true, nil
}

func (f *JobFactory) JobsByStatus(ctx context.Context, status status.Status) ([]Job, error) {
	queries := f.queries.Queries(ctx)

	jobs, err := queries.JobsByStatus(ctx, status.String())
	if err != nil {
		return nil, err
	}

	dbJobs := make([]Job, 0, len(jobs))
	for _, j := range jobs {
		dbJobs = append(dbJobs, newJob(j, f.queries))
	}

	return dbJobs, nil
}

func (f *JobFactory) WithTx(ctx context.Context, txFunc func(txCtx context.Context) error) error {
	return f.queries.WithTx(ctx, txFunc)
}

func newJob(j psql.Job, q *Queries) *job {
	return &job{Job: j, queries: q}
}

type Job interface {
	Transactor

	ID() string
	BuildID() string
	Name() string
	Status() status.Status
	Config() model.JobConfig
	Plan() model.TaskPlan
	Model() model.Job

	Lock(ctx context.Context) error

	Pending(ctx context.Context) error
	Start(ctx context.Context) error
	Abort(ctx context.Context) error
	Finish(ctx context.Context, status status.Status) error

	Tasks(ctx context.Context) ([]Task, error)
}

type job struct {
	psql.Job

	queries *Queries
}

func (j *job) ID() string {
	return j.Job.ID
}

func (j *job) BuildID() string {
	return j.Job.BuildID
}

func (j *job) Name() string {
	return j.Job.Name
}

func (j *job) Status() status.Status {
	return status.Status(j.Job.Status)
}

func (j *job) Config() model.JobConfig {
	var config model.JobConfig
	_ = json.Unmarshal(j.Job.Config, &config)
	return config
}

func (j *job) Plan() model.TaskPlan {
	var plan model.TaskPlan
	_ = json.Unmarshal(j.Job.Plan, &plan)
	return plan
}

func (j *job) Model() model.Job {
	return model.Job{
		ID:      j.ID(),
		BuildID: j.BuildID(),
		Name:    j.Name(),
		Status:  j.Status(),
		Config:  j.Config(),
		Plan:    j.Plan(),
	}
}

func (j *job) Lock(ctx context.Context) error {
	queries := j.queries.Queries(ctx)

	lockedJob, err := queries.LockJob(ctx, j.ID())
	if err != nil {
		return err
	}

	j.Job = lockedJob

	return nil
}

func (j *job) Pending(ctx context.Context) error {
	queries := j.queries.Queries(ctx)

	if j.Status().IsRunning() {
		return ErrAlreadyRunning
	}

	if j.Status().IsFinished() {
		return ErrAlreadyFinished
	}

	updatedJob, err := queries.UpdateJobStatus(
		ctx,
		psql.UpdateJobStatusParams{
			ID:     j.ID(),
			Status: status.StatusPending.String(),
		},
	)
	if err != nil {
		return err
	}

	j.Job = updatedJob

	return nil
}

func (j *job) Start(ctx context.Context) error {
	queries := j.queries.Queries(ctx)

	if j.Status().IsFinished() {
		return ErrAlreadyFinished
	}

	updatedJob, err := queries.UpdateJobStatus(
		ctx,
		psql.UpdateJobStatusParams{
			ID:     j.ID(),
			Status: status.StatusStarted.String(),
		},
	)
	if err != nil {
		return err
	}

	j.Job = updatedJob

	return nil
}

func (j *job) Finish(ctx context.Context, status status.Status) error {
	queries := j.queries.Queries(ctx)

	if j.Status().IsFinished() {
		return ErrAlreadyFinished
	}

	updatedJob, err := queries.UpdateJobStatus(
		ctx,
		psql.UpdateJobStatusParams{
			ID:     j.ID(),
			Status: status.String(),
		},
	)
	if err != nil {
		return err
	}

	j.Job = updatedJob

	return nil
}

func (j *job) Abort(ctx context.Context) error {
	return j.Finish(ctx, status.StatusAborted)
}

func (j *job) Tasks(ctx context.Context) ([]Task, error) {
	queries := j.queries.Queries(ctx)

	tasks, err := queries.TasksByJob(ctx, j.ID())
	if err != nil {
		return nil, err
	}

	dbTasks := make([]Task, 0, len(tasks))
	for _, t := range tasks {
		dbTasks = append(dbTasks, newTask(t, j.queries))
	}

	return dbTasks, nil
}

func (j *job) WithTx(ctx context.Context, txFunc func(txCtx context.Context) error) error {
	return j.queries.WithTx(ctx, txFunc)
}
