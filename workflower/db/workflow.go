package db

import (
	"context"
	"encoding/json"

	"github.com/sazonovItas/mini-ci/workflower/db/gen/psql"
	"github.com/sazonovItas/mini-ci/workflower/model"
)

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
