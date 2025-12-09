package runtime

import (
	"context"
	"fmt"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/errdefs"
)

type Task struct {
	task       containerd.Task
	exitStatus <-chan containerd.ExitStatus
}

func NewTask(task containerd.Task, ch <-chan containerd.ExitStatus) *Task {
	return &Task{
		task:       task,
		exitStatus: ch,
	}
}

func (t *Task) ID() string {
	return t.task.ID()
}

func (t *Task) Task() containerd.Task {
	return t.task
}

func (t *Task) WaitExitStatus(ctx context.Context) (int, error) {
	status := <-t.exitStatus

	if err := status.Error(); err != nil {
		return 0, fmt.Errorf("waiting for exit status: %w", err)
	}

	if err := t.task.CloseIO(ctx, containerd.WithStdinCloser); err != nil {
		return 0, fmt.Errorf("task close stdin: %w", err)
	}

	t.task.IO().Cancel()
	t.task.IO().Wait()

	_ = t.task.IO().Close()

	_, err := t.task.Delete(ctx)
	if err != nil && !errdefs.IsNotFound(err) {
		return 0, fmt.Errorf("delete task: %w", err)
	}

	return int(status.ExitCode()), nil
}
