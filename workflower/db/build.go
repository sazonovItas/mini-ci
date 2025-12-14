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

type BuildFactory struct {
	queries *Queries
}

func NewBuildFactory(queries *Queries) *BuildFactory {
	return &BuildFactory{queries: queries}
}

func (f *BuildFactory) New(ctx context.Context, b model.Build) (Build, error) {
	queries := f.queries.Queries(ctx)

	config, err := json.Marshal(b.Config)
	if err != nil {
		return nil, err
	}

	plan, err := json.Marshal(b.Plan)
	if err != nil {
		return nil, err
	}

	got, err := queries.CreateBuild(
		ctx,
		psql.CreateBuildParams{
			ID:         b.ID,
			WorkflowID: b.WorkflowID,
			Status:     status.StatusCreated.String(),
			Config:     config,
			Plan:       plan,
		})
	if err != nil {
		return nil, err
	}

	return newBuild(got, f.queries), nil
}

func (f *BuildFactory) Build(ctx context.Context, id string) (Build, bool, error) {
	queries := f.queries.Queries(ctx)

	got, err := queries.Build(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return newBuild(got, f.queries), true, nil
}

func (f *BuildFactory) BuildsByStatus(ctx context.Context, status status.Status) ([]Build, error) {
	queries := f.queries.Queries(ctx)

	builds, err := queries.BuildsByStatus(ctx, status.String())
	if err != nil {
		return nil, err
	}

	dbBuilds := make([]Build, 0, len(builds))
	for _, b := range builds {
		dbBuilds = append(dbBuilds, newBuild(b, f.queries))
	}

	return dbBuilds, nil
}

func (f *BuildFactory) WithTx(ctx context.Context, txFunc func(txCtx context.Context) error) error {
	return f.queries.WithTx(ctx, txFunc)
}

func newBuild(b psql.Build, q *Queries) *build {
	return &build{Build: b, queries: q}
}

type Build interface {
	Transactor

	ID() string
	WorkflowID() string
	Status() status.Status
	Config() model.WorkflowConfig
	Plan() model.JobPlan
	Model() model.Build

	SetPending(ctx context.Context) error

	Start(ctx context.Context) error
	Abort(ctx context.Context) error
	Finish(ctx context.Context, status status.Status) error

	Jobs(ctx context.Context) ([]Job, error)
}

type build struct {
	psql.Build

	queries *Queries
}

func (b *build) ID() string {
	return b.Build.ID
}

func (b *build) WorkflowID() string {
	return b.Build.WorkflowID
}

func (b *build) Status() status.Status {
	return status.Status(b.Build.Status)
}

func (b *build) Config() model.WorkflowConfig {
	var cfg model.WorkflowConfig
	_ = json.Unmarshal(b.Build.Config, &cfg)
	return cfg
}

func (b *build) Plan() model.JobPlan {
	var plan model.JobPlan
	_ = json.Unmarshal(b.Build.Plan, &plan)
	return plan
}

func (b *build) Model() model.Build {
	return model.Build{
		ID:         b.ID(),
		WorkflowID: b.WorkflowID(),
		Status:     b.Status(),
		Config:     b.Config(),
		Plan:       b.Plan(),
	}
}

func (b *build) SetPending(ctx context.Context) error {
	return b.WithTx(ctx, func(txCtx context.Context) error {
		queries := b.queries.Queries(txCtx)

		lockedBuild, err := queries.LockBuild(txCtx, b.ID())
		if err != nil {
			return err
		}
		b.Build = lockedBuild

		if b.IsFinished() {
			return ErrAlreadyFinished
		}

		updatedBuild, err := queries.UpdateBuildStatus(
			txCtx,
			psql.UpdateBuildStatusParams{
				ID:     b.ID(),
				Status: status.StatusPending.String(),
			},
		)
		if err != nil {
			return err
		}
		b.Build = updatedBuild

		return nil
	})
}

func (b *build) Start(ctx context.Context) error {
	return b.WithTx(ctx, func(txCtx context.Context) error {
		queries := b.queries.Queries(txCtx)

		lockedBuild, err := queries.LockBuild(txCtx, b.ID())
		if err != nil {
			return err
		}
		b.Build = lockedBuild

		if b.IsRunning() {
			return ErrAlreadyRunning
		}

		if b.IsFinished() {
			return ErrAlreadyFinished
		}

		updatedBuild, err := queries.UpdateBuildStatus(
			txCtx,
			psql.UpdateBuildStatusParams{
				ID:     b.ID(),
				Status: status.StatusStarted.String(),
			},
		)
		if err != nil {
			return err
		}
		b.Build = updatedBuild

		return nil
	})
}

func (b *build) Abort(ctx context.Context) error {
	return b.Finish(ctx, status.StatusAborted)
}

func (b *build) Finish(ctx context.Context, status status.Status) error {
	return b.WithTx(ctx, func(txCtx context.Context) error {
		queries := b.queries.Queries(txCtx)

		lockedBuild, err := queries.LockBuild(txCtx, b.ID())
		if err != nil {
			return err
		}
		b.Build = lockedBuild

		if !b.IsRunning() {
			return ErrIsNotRunning
		}

		if b.IsFinished() {
			return ErrAlreadyFinished
		}

		updatedBuild, err := queries.UpdateBuildStatus(
			txCtx,
			psql.UpdateBuildStatusParams{
				ID:     b.ID(),
				Status: string(status),
			},
		)
		if err != nil {
			return err
		}
		b.Build = updatedBuild

		return nil
	})
}

func (b *build) Jobs(ctx context.Context) ([]Job, error) {
	queries := b.queries.Queries(ctx)

	jobs, err := queries.JobsByBuild(ctx, b.ID())
	if err != nil {
		return nil, err
	}

	dbJobs := make([]Job, 0, len(jobs))
	for _, j := range jobs {
		dbJobs = append(dbJobs, newJob(j, b.queries))
	}

	return dbJobs, nil
}

func (b *build) WithTx(ctx context.Context, f func(txCtx context.Context) error) error {
	return b.queries.WithTx(ctx, f)
}

func (b *build) IsRunning() bool {
	return b.Status().IsRunning()
}

func (b *build) IsFinished() bool {
	return b.Status().IsFinished()
}
