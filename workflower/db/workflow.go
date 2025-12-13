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

func (w *workflow) WithTx(ctx context.Context, txFunc func(txCtx context.Context) error) error {
	return w.queries.WithTx(ctx, txFunc)
}
