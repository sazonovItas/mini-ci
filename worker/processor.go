package worker

import (
	"context"
	"fmt"
	"sync"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
	runtimespec "github.com/sazonovItas/mini-ci/worker/runtime"
)

type EventLogger interface {
	Log(message string)
	Process(id string, stdout <-chan string, stderr <-chan string) error
}

type EventLoggerFactory interface {
	New(origin events.EventOrigin) EventLogger
}

type Sender interface {
	Send(event events.Event)
}

type Runtime interface {
	Pull(ctx context.Context, imageRef string) error
	Container(ctx context.Context, id string) (*runtimespec.Container, error)
	Create(ctx context.Context, spec runtimespec.ContainerConfig) (*runtimespec.Container, error)
	Destroy(ctx context.Context, id string) error
}

type EventProcessor struct {
	wg sync.WaitGroup

	sender        Sender
	runtime       Runtime
	loggerFactory EventLoggerFactory
}

func NewEventProcessor(sender Sender, runtime Runtime) *EventProcessor {
	return &EventProcessor{
		sender:        sender,
		runtime:       runtime,
		loggerFactory: NewEventLoggerFactory(sender),
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
					p.sender.Send(events.NewErrorEvent(event.Origin(), err.Error()))
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
	case events.StartInitContainer:
		return p.startInitContainer(ctx, event)
	case events.StartScript:
		return p.startScript(ctx, event)
	default:
		log.G(ctx).Errorf("cannot process event: %s", event.Type())
	}

	return nil
}

func (p *EventProcessor) startInitContainer(ctx context.Context, event events.StartInitContainer) error {
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

	finishInitContainer := events.FinishInitContainer{
		EventOrigin: event.Origin(),
		ContainerID: container.ID(),
	}
	p.sender.Send(finishInitContainer)

	return nil
}

func (p *EventProcessor) startScript(ctx context.Context, event events.StartScript) error {
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

	task, err := container.Start(ctx, evLogger)
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

	finishScript := events.FinishScript{
		EventOrigin: event.Origin(),
		ExitStatus:  exitStatus,
		Succeeded:   exitStatus != 0,
	}
	p.sender.Send(finishScript)

	return nil
}
