package worker

import (
	"context"
	"fmt"
	"sync"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/worker/config"
	"github.com/sazonovItas/mini-ci/worker/runtime"
)

type Worker struct {
	runtime *runtime.Runtime

	socketIORunner *SocketIORunner
	eventProcessor *EventProcessor

	wg sync.WaitGroup
}

func New(cfg config.Config) (*Worker, error) {
	ctrRuntime, err := newContainerRuntime(cfg.Runtime)
	if err != nil {
		return nil, fmt.Errorf("new runtime: %w", err)
	}

	sockerIORunner := NewSocketIORunner(
		cfg.Name,
		cfg.SocketIO.URL,
		cfg.SocketIO.Namespace,
		cfg.SocketIO.EventName,
	)

	eventProcessor := NewEventProcessor(ctrRuntime, sockerIORunner)

	worker := &Worker{
		runtime:        ctrRuntime,
		socketIORunner: sockerIORunner,
		eventProcessor: eventProcessor,
	}

	return worker, nil
}

func (w *Worker) Start(ctx context.Context) error {
	w.wg.Go(func() {
		evch, closer := w.socketIORunner.Events()
		defer func() {
			_ = closer.Close()
		}()

		w.eventProcessor.Process(ctx, evch)
	})

	if err := w.socketIORunner.Start(ctx); err != nil {
		log.G(ctx).WithError(err).Error("failed to start socker io runner")
		return err
	}

	return nil
}

func (w *Worker) Stop(ctx context.Context) error {
	w.wg.Go(func() {
		if err := w.socketIORunner.Stop(ctx); err != nil {
			log.G(ctx).WithError(err).Error("failed to stop socket io runner")
		}
	})

	w.wg.Go(func() {
		w.eventProcessor.Wait()
	})

	w.wg.Wait()

	return nil
}

func newContainerRuntime(cfg config.RuntimeConfig) (*runtime.Runtime, error) {
	client, err := containerd.New(cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("new containerd client: %w", err)
	}

	r, err := runtime.New(
		client,
		runtime.WithSnapshotter(cfg.Snapshotter),
		runtime.WithDataStorePath(cfg.DataStorePath),
	)
	if err != nil {
		return nil, fmt.Errorf("new container runtime: %w", err)
	}

	return r, nil
}
