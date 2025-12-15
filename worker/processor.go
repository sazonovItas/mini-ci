package worker

import (
	"context"
	"fmt"
	"sync"

	"github.com/containerd/errdefs"
	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
	runtimespec "github.com/sazonovItas/mini-ci/worker/runtime"
	"github.com/sazonovItas/mini-ci/worker/runtime/logging"
)

type EventLogger interface {
	Log(msg string)
	Process(id string, stdout <-chan string, stderr <-chan string) error
}

type EventLoggerFactory interface {
	New(origin events.EventOrigin) EventLogger
}

type Runtime interface {
	Pull(ctx context.Context, imageRef string) error
	Container(ctx context.Context, id string) (*runtimespec.Container, error)
	Create(ctx context.Context, spec runtimespec.ContainerConfig) (*runtimespec.Container, error)
	Destroy(ctx context.Context, id string) error
}

type EventProcessor struct {
	wg sync.WaitGroup

	runtime       Runtime
	Publisher     events.Publisher
	loggerFactory EventLoggerFactory

	defaultLogger logging.Logger
}

func NewEventProcessor(runtime Runtime, publisher events.Publisher) *EventProcessor {
	return &EventProcessor{
		runtime:       runtime,
		Publisher:     publisher,
		loggerFactory: NewEventLoggerFactory(publisher),
		defaultLogger: logging.NewJSONLogger("/var/lib/minici"),
	}
}

func (p *EventProcessor) Process(ctx context.Context, evch <-chan events.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-evch:
			if !ok {
				return
			}

			p.wg.Go(func() {
				if err := p.process(ctx, event); err != nil {
					_ = p.Publisher.Publish(ctx, events.NewErrorEvent(event.Origin(), err.Error()))
				}
			})
		}
	}
}

func (p *EventProcessor) Wait() {
	p.wg.Wait()
}

func (p *EventProcessor) process(ctx context.Context, ev events.Event) error {
	switch event := ev.(type) {
	case events.InitContainerStart:
		return p.containerInitStart(ctx, event)

	case events.ScriptStart:
		return p.scriptStart(ctx, event)

	case events.TaskAbort:
		return p.scriptAbort(ctx, event)

	case events.CleanupContainer:
		return p.containerDestroy(ctx, event)

	default:
		log.G(ctx).Errorf("cannot process event: %s", event.Type())
	}

	return nil
}

func (p *EventProcessor) containerInitStart(ctx context.Context, event events.InitContainerStart) error {
	evLogger := p.loggerFactory.New(event.Origin())

	if err := p.runtime.Pull(ctx, event.Config.Image); err != nil {
		evLogger.Log(fmt.Sprintf("failed to pull image %s", event.Config.Image))
		return err
	}

	evLogger.Log(fmt.Sprintf("successfully pulled image %s", event.Config.Image))

	ctrConfig := runtimespec.ContainerConfig{
		Image: event.Config.Image,
		Cwd:   event.Config.Cwd,
		Env:   event.Config.Env,
	}

	container, err := p.runtime.Create(ctx, ctrConfig)
	if err != nil {
		evLogger.Log("failed to create container")
		return err
	}

	evLogger.Log(fmt.Sprintf("created container %s", container.ShortID()))

	finishInitContainer := events.InitContainerFinish{
		EventOrigin: event.Origin(),
		ContainerID: container.ID(),
	}
	_ = p.Publisher.Publish(ctx, finishInitContainer)

	return nil
}

func (p *EventProcessor) scriptStart(ctx context.Context, event events.ScriptStart) error {
	evLogger := p.loggerFactory.New(event.Origin())

	container, err := p.runtime.Container(ctx, event.Config.ContainerID)
	if err != nil {
		return err
	}

	taskConfig := runtimespec.TaskConfig{
		Command: event.Config.Command,
		Args:    event.Config.Args,
	}
	if err := container.NewTask(ctx, taskConfig); err != nil {
		evLogger.Log("failed to create task")
		return err
	}

	evLogger.Log("created script task")

	task, err := container.Start(ctx, evLogger, p.defaultLogger)
	if err != nil {
		evLogger.Log("failed to start task")
		return err
	}

	evLogger.Log("successfully started task")

	exitStatus, err := task.WaitExitStatus(ctx)
	if err != nil {
		evLogger.Log("failed to wait exit status")
		return err
	}

	finishScript := events.ScriptFinish{
		EventOrigin: event.Origin(),
		ExitStatus:  exitStatus,
		Succeeded:   exitStatus != 0,
	}
	_ = p.Publisher.Publish(ctx, finishScript)

	return nil
}

func (p *EventProcessor) scriptAbort(ctx context.Context, event events.TaskAbort) error {
	container, err := p.runtime.Container(ctx, event.ContainerID)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil
		}

		return err
	}

	if err := container.Stop(ctx, false); err != nil {
		return err
	}

	return nil
}

func (p *EventProcessor) containerDestroy(ctx context.Context, event events.CleanupContainer) error {
	if err := p.runtime.Destroy(ctx, event.ContainerID); err != nil {
		log.G(ctx).WithError(err).Errorf("failed to destroy container %s", event.ContainerID)
	}

	return nil
}
