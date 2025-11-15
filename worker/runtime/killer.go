package runtime

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"time"

	containerd "github.com/containerd/containerd/v2/client"
)

const (
	gracefulSignal = syscall.SIGTERM

	ungracefulSignal = syscall.SIGKILL

	gracefulKillTimeout = 15 * time.Second

	killTimeout = 10 * time.Second
)

type TaskKiller interface {
	Kill(ctx context.Context, task containerd.Task, signal syscall.Signal, waitTimeout time.Duration) error
}

type killer struct {
	taskKiller          TaskKiller
	killTimeout         time.Duration
	gracefulKillTimeout time.Duration
}

var _ Killer = (*killer)(nil)

func NewKiller() *killer {
	return &killer{
		taskKiller:          NewTaskKiller(),
		killTimeout:         killTimeout,
		gracefulKillTimeout: gracefulKillTimeout,
	}
}

func (k killer) Kill(ctx context.Context, task containerd.Task, gracefully bool) error {
	if gracefully {
		err := k.taskKiller.Kill(ctx, task, gracefulSignal, k.gracefulKillTimeout)
		if err == nil || !errors.Is(err, ErrTaskKillTimeout) {
			return fmt.Errorf("gracefully kill task: %w", err)
		}
	}

	err := k.taskKiller.Kill(ctx, task, ungracefulSignal, k.killTimeout)
	if err != nil {
		return fmt.Errorf("kill task: %w", err)
	}

	return nil
}
