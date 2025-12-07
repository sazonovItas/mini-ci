package workflower

import (
	"context"

	"github.com/sazonovItas/mini-ci/workflower/config"
)

type Workflower struct{}

func New(_ config.Config) *Workflower {
	return &Workflower{}
}

func (w Workflower) Start(ctx context.Context) error {
	return nil
}

func (w Workflower) Stop(ctx context.Context) error {
	return nil
}
