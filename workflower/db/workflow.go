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

type WorkflowRepository struct {
	queries *Queries
}

func NewWorkflowRepository(queries *Queries) *WorkflowRepository {
	return &WorkflowRepository{queries: queries}
}

func (r *WorkflowRepository) Workflows(ctx context.Context, offset, limit int) ([]model.Workflow, error) {
	const (
		defaultLimit = 10
	)

	queries := r.queries.Queries(ctx)

	if limit == 0 {
		limit = defaultLimit
	}

	dbWorkflows, err := queries.Workflows(
		ctx,
		psql.WorkflowsParams{
			Offset: int32(offset),
			Limit:  int32(limit),
		},
	)
	if err != nil {
		return nil, err
	}

	workflows := make([]model.Workflow, 0, len(dbWorkflows))
	for _, w := range dbWorkflows {
		var config model.WorkflowConfig
		if err := json.Unmarshal(w.Config, &config); err != nil {
			return nil, err
		}

		workflow := model.Workflow{
			ID:     w.ID,
			Name:   w.Name,
			Config: config,
		}

		workflows = append(workflows, workflow)
	}

	err = r.queries.WithTx(ctx, func(txCtx context.Context) error {
		queries := r.queries.Queries(txCtx)

		for i := range workflows {
			build, err := queries.Build(txCtx, dbWorkflows[i].CurrBuildID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					continue
				}

				return err
			}

			workflows[i].CurrBuild = &model.Build{
				ID:         build.ID,
				WorkflowID: build.WorkflowID,
				Status:     status.Status(build.Status),
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return workflows, nil
}

func (r *WorkflowRepository) Delete(ctx context.Context, id string) error {
	queries := r.queries.Queries(ctx)

	if err := queries.DeleteWorkflow(ctx, id); err != nil {
		return err
	}

	return nil
}

type WorkflowFactory struct {
	queries *Queries
}

func NewWorkflowFactory(queries *Queries) *WorkflowFactory {
	return &WorkflowFactory{queries: queries}
}

func (f *WorkflowFactory) New(ctx context.Context, w model.Workflow) (Workflow, error) {
	queries := f.queries.Queries(ctx)

	config, err := json.Marshal(w.Config)
	if err != nil {
		return nil, err
	}

	got, err := queries.CreateWorkflow(
		ctx,
		psql.CreateWorkflowParams{
			ID:     w.ID,
			Name:   w.Name,
			Config: config,
		},
	)
	if err != nil {
		return nil, err
	}

	return newWorkflow(got, f.queries), nil
}

func (f *WorkflowFactory) Workflow(ctx context.Context, id string) (Workflow, error) {
	queries := f.queries.Queries(ctx)

	got, err := queries.Workflow(ctx, id)
	if err != nil {
		return nil, err
	}

	return newWorkflow(got, f.queries), nil
}

func (f *WorkflowFactory) WorkflowByName(ctx context.Context, name string) (Workflow, error) {
	queries := f.queries.Queries(ctx)

	got, err := queries.WorkflowByName(ctx, name)
	if err != nil {
		return nil, err
	}

	return newWorkflow(got, f.queries), nil
}

func (f *WorkflowFactory) WithTx(ctx context.Context, txFunc func(txCtx context.Context) error) error {
	return f.queries.WithTx(ctx, txFunc)
}

func newWorkflow(w psql.Workflow, q *Queries) *workflow {
	return &workflow{Workflow: w, queries: q}
}

type Workflow interface {
	Transactor

	ID() string
	Name() string
	CurrBuildID() string
	Config() model.WorkflowConfig
	Model() model.Workflow

	Update(ctx context.Context, workflow model.Workflow) error
	UpdateCurrentBuild(ctx context.Context, buildID string) error
	Builds(ctx context.Context) ([]Build, error)
}

type workflow struct {
	psql.Workflow

	queries *Queries
}

func (w *workflow) ID() string {
	return w.Workflow.ID
}

func (w *workflow) Name() string {
	return w.Workflow.Name
}

func (w *workflow) CurrBuildID() string {
	return w.Workflow.CurrBuildID
}

func (w *workflow) Config() model.WorkflowConfig {
	var config model.WorkflowConfig
	_ = json.Unmarshal(w.Workflow.Config, &config)
	return config
}

func (w *workflow) Model() model.Workflow {
	return model.Workflow{
		ID:     w.ID(),
		Name:   w.Name(),
		Config: w.Config(),
	}
}

func (w *workflow) Update(ctx context.Context, workflow model.Workflow) error {
	queries := w.queries.Queries(ctx)

	config, err := json.Marshal(workflow.Config)
	if err != nil {
		return err
	}

	updatedWorkflow, err := queries.UpdateWorkflow(
		ctx,
		psql.UpdateWorkflowParams{
			ID:     w.ID(),
			Name:   workflow.Name,
			Config: config,
		},
	)
	if err != nil {
		return err
	}

	w.Workflow = updatedWorkflow

	return err
}

func (w *workflow) UpdateCurrentBuild(ctx context.Context, buildID string) error {
	queries := w.queries.Queries(ctx)

	updatedWorkflow, err := queries.UpdateWorkflowCurrentBuild(
		ctx,
		psql.UpdateWorkflowCurrentBuildParams{
			ID:          w.ID(),
			CurrBuildID: buildID,
		},
	)
	if err != nil {
		return err
	}

	w.Workflow = updatedWorkflow

	return err
}

func (w *workflow) CurrBuild(ctx context.Context) (Build, bool, error) {
	queries := w.queries.Queries(ctx)

	dbBuild, err := queries.Build(ctx, w.CurrBuildID())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, err
	}

	b := newBuild(dbBuild, w.queries)

	return b, true, nil
}

func (w *workflow) Builds(ctx context.Context) ([]Build, error) {
	queries := w.queries.Queries(ctx)

	builds, err := queries.BuildsByWorkflow(ctx, w.ID())
	if err != nil {
		return nil, err
	}

	var dbBuilds []Build
	for _, b := range builds {
		dbBuilds = append(dbBuilds, newBuild(b, w.queries))
	}

	return dbBuilds, nil
}

func (w *workflow) WithTx(ctx context.Context, txFunc func(txCtx context.Context) error) error {
	return w.queries.WithTx(ctx, txFunc)
}
