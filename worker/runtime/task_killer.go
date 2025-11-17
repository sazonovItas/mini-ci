package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"

	containerd "github.com/containerd/containerd/v2/client"
)

type taskKiller struct{}

var _ TaskKiller = (*taskKiller)(nil)

func NewTaskKiller() *taskKiller {
	return &taskKiller{}
}

func (k taskKiller) Kill(ctx context.Context, task containerd.Task, signal syscall.Signal, waitTimeout time.Duration) error {
	waitCtx, cancel := context.WithTimeout(ctx, waitTimeout)
	defer cancel()

	taskWaitStatus, err := task.Wait(waitCtx)
	if err != nil {
		return fmt.Errorf("task wait: %w", err)
	}

	if err := task.Kill(waitCtx, signal); err != nil {
		return fmt.Errorf("task kill signal %d: %w", signal, err)
	}

	select {
	case <-waitCtx.Done():
		if err := waitCtx.Err(); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return ErrTaskKillTimeout
			}

			return fmt.Errorf("wait ctx done: %w", err)
		}
	case exitStatus := <-taskWaitStatus:
		if err := exitStatus.Error(); err != nil {
			if strings.Contains(err.Error(), "deadline exceeded") {
				return ErrTaskKillTimeout
			}

			return fmt.Errorf("task exit status: %w", err)
		}
	}

	return nil
}
