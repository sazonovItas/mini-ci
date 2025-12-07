package worker

import (
	"context"
	"fmt"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/sazonovItas/mini-ci/worker/config"
	"github.com/sazonovItas/mini-ci/worker/runtime"
)

type Worker struct {
	runtime *runtime.Runtime
}

func New(cfg config.Config) (*Worker, error) {
	ctrRuntime, err := newContainerRuntime(cfg.Runtime)
	if err != nil {
		return nil, fmt.Errorf("new runtime: %w", err)
	}

	worker := &Worker{
		runtime: ctrRuntime,
	}

	return worker, nil
}

func (w *Worker) Start(ctx context.Context) error {
	return nil
}

func (w *Worker) Stop(ctx context.Context) error {
	if err := w.cleanup(); err != nil {
		return fmt.Errorf("clean up worker: %w", err)
	}

	return nil
}

func (w *Worker) cleanup() error {
	return nil
}

func newContainerRuntime(cfg config.RuntimeConfig) (*runtime.Runtime, error) {
	client, err := containerd.New(cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("new containerd client: %w", err)
	}

	r, err := runtime.New(client,
		runtime.WithSnapshotter(cfg.Snapshotter),
		runtime.WithDataStorePath(cfg.DataStorePath),
	)
	if err != nil {
		return nil, fmt.Errorf("new container runtime: %w", err)
	}

	return r, nil
}
