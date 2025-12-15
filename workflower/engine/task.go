package engine

import (
	"context"
	"sync"
	"time"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/core/status"
	"github.com/sazonovItas/mini-ci/workflower/db"
	"github.com/sazonovItas/mini-ci/workflower/model"
)

type TaskWatcher struct {
	bus   events.Bus
	tasks *db.TaskFactory

	wg sync.WaitGroup

	ctx    context.Context
	cancel context.CancelFunc
}

func NewTaskWatcher(bus events.Bus, tasks *db.TaskFactory) *TaskWatcher {
	return &TaskWatcher{
		bus:   bus,
		tasks: tasks,
	}
}

func (w *TaskWatcher) Start(ctx context.Context) {
	w.ctx, w.cancel = context.WithCancel(ctx)

	w.wg.Go(func() { w.watchEvents(ctx) })
}

func (w *TaskWatcher) Stop(ctx context.Context) {
	w.cancel()

	w.wg.Wait()
}

func (w *TaskWatcher) watchEvents(ctx context.Context) {
	evch, errch := w.bus.Subscribe(
		ctx,
		events.WithEventTypes(
			events.EventTypeTaskStatus,
			events.EventTypeInitContainerFinish,
			events.EventTypeScriptFinish,
			events.EventTypeError,
		),
	)

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-evch:
			if !ok {
				return
			}

			err := w.processEvent(ctx, event)
			if err != nil {
				log.G(ctx).WithError(err).Error("task watcher: failed to process event")
			}

		case err := <-errch:
			if err != nil {
				log.G(ctx).WithError(err).Error("task watcher: failed to read events from bus")
			}
		}
	}
}

func (w *TaskWatcher) processEvent(ctx context.Context, ev events.Event) (err error) {
	switch event := ev.(type) {
	case events.InitContainerFinish:
		err = w.initContainerFinish(ctx, event)

	case events.ScriptFinish:
		err = w.scriptFinish(ctx, event)

	case events.Error:
		err = w.error(ctx, event)

	case events.TaskStatus:
		err = w.pendingStatus(ctx, event)

	default:
		log.G(ctx).Errorf("task watcher: unknown event type %T to process", event)
	}

	return err
}

func (w *TaskWatcher) initContainerFinish(ctx context.Context, event events.InitContainerFinish) error {
	return w.tasks.WithTx(ctx, func(txCtx context.Context) error {
		task, found, err := w.tasks.Task(txCtx, event.Origin().ID)
		if err != nil {
			return err
		}

		if !found {
			return nil
		}

		config := task.Config().Config.(model.InitStep)
		config.Outputs = &model.InitOutputs{ContainerID: event.ContainerID}

		err = task.UpdateConfig(ctx, model.Step{Config: config})
		if err != nil {
			return err
		}

		finishStatus := status.StatusSucceeded

		err = task.Finish(txCtx, finishStatus)
		if err != nil {
			return err
		}

		err = w.publishStatusChanged(txCtx, task.Model())
		if err != nil {
			return err
		}

		return nil
	})
}

func (w *TaskWatcher) scriptFinish(ctx context.Context, event events.ScriptFinish) error {
	return w.tasks.WithTx(ctx, func(txCtx context.Context) error {
		task, found, err := w.tasks.Task(txCtx, event.Origin().ID)
		if err != nil {
			return err
		}

		if !found {
			return nil
		}

		config := task.Config().Config.(model.ScriptStep)
		config.Outputs = &model.ScriptOutputs{ExitStatus: event.ExitStatus, Succeeded: event.Succeeded}

		err = task.UpdateConfig(ctx, model.Step{Config: config})
		if err != nil {
			return err
		}

		finishStatus := status.StatusSucceeded
		if !event.Succeeded {
			finishStatus = status.StatusFailed
		}

		err = task.Finish(txCtx, finishStatus)
		if err != nil {
			return err
		}

		err = w.publishStatusChanged(txCtx, task.Model())
		if err != nil {
			return err
		}

		return nil
	})
}

func (w *TaskWatcher) error(ctx context.Context, event events.Error) error {
	return w.tasks.WithTx(ctx, func(txCtx context.Context) error {
		task, found, err := w.tasks.Task(txCtx, event.Origin().ID)
		if err != nil {
			return err
		}

		if !found {
			return nil
		}

		err = task.Finish(txCtx, status.StatusErrored)
		if err != nil {
			return err
		}

		err = w.publishStatusChanged(txCtx, task.Model())
		if err != nil {
			return err
		}

		return nil
	})
}

func (w *TaskWatcher) pendingStatus(ctx context.Context, event events.TaskStatus) error {
	if !event.Status.IsPending() {
		return nil
	}

	return w.tasks.WithTx(ctx, func(txCtx context.Context) error {
		task, found, err := w.tasks.Task(txCtx, event.Origin().ID)
		if err != nil {
			return err
		}

		if !found {
			return nil
		}

		nextStatus, event := w.startTask(ctx, task.Model())

		if nextStatus.IsFinished() {
			err = task.Finish(txCtx, nextStatus)
			if err != nil {
				return err
			}
		} else {
			err = task.Start(txCtx)
			if err != nil {
				return err
			}
		}

		if err := w.bus.Publish(ctx, event); err != nil {
			return err
		}

		if err := w.publishStatusChanged(ctx, task.Model()); err != nil {
			return err
		}

		return nil
	})
}

func (w *TaskWatcher) startTask(ctx context.Context, task model.Task) (status.Status, events.Event) {
	var (
		origin     = events.NewEventOrigin(task.ID)
		nextStatus = status.StatusStarted
		event      events.Event
	)

	switch step := task.Config.Config.(type) {
	case model.InitStep:
		event = events.InitContainerStart{
			EventOrigin: origin,
			Config: events.ContainerConfig{
				Image: step.Image,
				Cwd:   step.Cwd,
				Env:   step.Env,
			},
		}

	case model.ScriptStep:
		event = events.ScriptStart{
			EventOrigin: origin,
			Config: events.ScriptConfig{
				ContainerID: step.ContainerID,
				Command:     step.Command,
				Args:        step.Args,
			},
		}

	case model.CleanupStep:
		event = events.CleanupContainer{
			EventOrigin: origin,
			ContainerID: step.ContainerID,
		}

		nextStatus = status.StatusSucceeded

	default:
		log.G(ctx).Errorf("task watcher unknown task type %T", step)
	}

	return nextStatus, event
}

func (w *TaskWatcher) publishStatusChanged(ctx context.Context, task model.Task) error {
	event := events.TaskStatus{
		ChangeStatus: events.ChangeStatus{
			EventOrigin: events.EventOrigin{
				ID:        task.ID,
				OccuredAt: time.Now().UTC(),
			},
			Status: task.Status,
		},
		JobID: task.JobID,
	}

	return w.bus.Publish(ctx, event)
}
