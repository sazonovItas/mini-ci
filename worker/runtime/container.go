package runtime

import (
	"context"
	"fmt"
	"io"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/errdefs"
)

type TaskIO struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type Container struct {
	container containerd.Container
	ioManager IOManager
	killer    Killer
}

func NewContainer(
	container containerd.Container,
	ioManager IOManager,
	killer Killer,
) *Container {
	return &Container{
		container: container,
		ioManager: ioManager,
		killer:    killer,
	}
}

func (c *Container) ID() string {
	return c.container.ID()
}

func (c *Container) Container() containerd.Container {
	return c.container
}

func (c *Container) Start(ctx context.Context) (*Task, error) {
	task, err := c.container.Task(ctx, cio.Load)
	if err != nil {
		return nil, fmt.Errorf("retrieve task: %w", err)
	}

	if err := task.Start(ctx); err != nil {
		return nil, fmt.Errorf("task start: %w", err)
	}

	taskWaitStatus, err := task.Wait(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("task wait: %w", err)
	}

	return NewTask(task, taskWaitStatus), nil
}

func (c *Container) Attach(ctx context.Context, taskIO TaskIO) (*Task, error) {
	ioAttach := c.ioManager.Attach(
		c.ID(),
		cio.NewAttach(containerCIO(taskIO, false)...),
	)

	task, err := c.container.Task(ctx, ioAttach)
	if err != nil {
		return nil, fmt.Errorf("retrieve task: %w", err)
	}

	status, err := task.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("task status: %w", err)
	}

	if status.Status != containerd.Running {
		return nil, fmt.Errorf("task is not running: status = %s", status.Status)
	}

	waitTaskStatus, err := task.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("task wait: %w", err)
	}

	return NewTask(task, waitTaskStatus), nil
}

func (c *Container) Stop(ctx context.Context, kill bool) error {
	task, err := c.container.Task(ctx, cio.Load)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("retrieve task: %w", err)
	}

	if err := c.killer.Kill(ctx, task, !kill); err != nil {
		return fmt.Errorf("kill task: %w", err)
	}

	return nil
}

func (c *Container) newTask(ctx context.Context, taskIO TaskIO) (containerd.Task, error) {
	ioCreator := c.ioManager.Creator(
		c.ID(),
		cio.NewCreator(containerCIO(taskIO, false)...),
	)

	task, err := c.container.NewTask(ctx, ioCreator)
	if err != nil {
		return nil, fmt.Errorf("create new task: %w", err)
	}

	if err := task.CloseIO(ctx, containerd.WithStdinCloser); err != nil {
		return nil, fmt.Errorf("close stdin stream: %w", err)
	}

	return task, nil
}

func containerCIO(taskIO TaskIO, tty bool) []cio.Opt {
	if !tty {
		return []cio.Opt{
			cio.WithStreams(
				taskIO.Stdin,
				taskIO.Stdout,
				taskIO.Stderr,
			),
		}
	}

	opts := []cio.Opt{
		cio.WithStreams(
			taskIO.Stdin,
			taskIO.Stdout,
			taskIO.Stderr,
		),
		cio.WithTerminal,
	}

	return opts
}
