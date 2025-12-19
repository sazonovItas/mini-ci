package engine

import (
	"context"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/core/status"
	"github.com/sazonovItas/mini-ci/workflower/db"
	"github.com/sazonovItas/mini-ci/workflower/model"
)

type TaskProcessor struct {
	tasks     *db.TaskFactory
	publisher events.Publisher
}

func NewTaskProcessor(
	tasks *db.TaskFactory,
	publisher events.Publisher,
) TaskProcessor {
	return TaskProcessor{tasks: tasks, publisher: publisher}
}

func (p TaskProcessor) Filters() []events.FilterFunc {
	return []events.FilterFunc{
		events.WithEventTypes(
			events.EventTypeTaskStatus,
			events.EventTypeInitContainerFinish,
			events.EventTypeScriptFinish,
			events.EventTypeError,
			events.EventTypeTaskAbort,
		),
	}
}

func (p TaskProcessor) ProcessEvent(ctx context.Context, event events.Event) (err error) {
	switch e := event.(type) {
	case events.TaskStatus:
		err = p.taskStatus(ctx, e)

	case events.InitContainerFinish:
		err = p.initContainerFinish(ctx, e)

	case events.ScriptFinish:
		err = p.scriptFinish(ctx, e)

	case events.TaskError:
		err = p.error(ctx, e)

	case events.TaskAbort:
		err = p.taskAbort(ctx, e)

	default:
		log.G(ctx).Errorf("task processor: unknown event type %T to process", e)
	}

	if err != nil {
		log.G(ctx).WithError(err).Errorf("task processor: failed to process event %T", event)
	}

	return err
}

func (p TaskProcessor) initContainerFinish(ctx context.Context, event events.InitContainerFinish) error {
	return p.tasks.WithTx(ctx, func(txCtx context.Context) error {
		task, found, err := p.tasks.Task(txCtx, event.Origin().ID)
		if err != nil {
			return err
		}

		if !found {
			return ErrTaskNotFound
		}

		err = task.Lock(txCtx)
		if err != nil {
			return err
		}

		config := task.Config().(*model.InitStep)
		config.Outputs = &model.InitOutputs{ContainerID: event.ContainerID}

		err = task.UpdateConfig(ctx, config)
		if err != nil {
			return err
		}

		finishStatus := status.StatusSucceeded

		err = task.Finish(txCtx, finishStatus)
		if err != nil {
			return err
		}

		err = p.publishStatusChanged(txCtx, task.Model())
		if err != nil {
			return err
		}

		return nil
	})
}

func (p TaskProcessor) scriptFinish(ctx context.Context, event events.ScriptFinish) error {
	return p.tasks.WithTx(ctx, func(txCtx context.Context) error {
		task, found, err := p.tasks.Task(txCtx, event.Origin().ID)
		if err != nil {
			return err
		}

		if !found {
			return ErrTaskNotFound
		}

		err = task.Lock(txCtx)
		if err != nil {
			return err
		}

		config := task.Config().(*model.ScriptStep)
		config.Outputs = &model.ScriptOutputs{ExitStatus: event.ExitStatus, Succeeded: event.Succeeded}

		err = task.UpdateConfig(ctx, config)
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

		err = p.publishStatusChanged(txCtx, task.Model())
		if err != nil {
			return err
		}

		return nil
	})
}

func (p TaskProcessor) error(ctx context.Context, event events.TaskError) error {
	return p.tasks.WithTx(ctx, func(txCtx context.Context) error {
		task, found, err := p.tasks.Task(ctx, event.Origin().ID)
		if err != nil {
			return err
		}

		if !found {
			return ErrTaskNotFound
		}

		err = task.Lock(txCtx)
		if err != nil {
			return err
		}

		err = task.Finish(txCtx, status.StatusErrored)
		if err != nil {
			return err
		}

		err = p.publishStatusChanged(txCtx, task.Model())
		if err != nil {
			return err
		}

		return nil
	})
}

func (p TaskProcessor) taskStatus(ctx context.Context, event events.TaskStatus) error {
	if !event.Status.IsPending() {
		return nil
	}

	return p.tasks.WithTx(ctx, func(txCtx context.Context) error {
		task, found, err := p.tasks.Task(ctx, event.Origin().ID)
		if err != nil {
			return err
		}

		if !found {
			return ErrTaskNotFound
		}

		err = task.Lock(txCtx)
		if err != nil {
			return err
		}

		nextStatus, event, err := p.startTask(ctx, task.Model())
		if err != nil {
			return err
		}

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

		if err := p.publish(txCtx, event); err != nil {
			return err
		}

		if err := p.publishStatusChanged(txCtx, task.Model()); err != nil {
			return err
		}

		return nil
	})
}

func (p TaskProcessor) startTask(ctx context.Context, task model.Task) (status.Status, events.Event, error) {
	var (
		origin     = events.NewEventOrigin(task.ID)
		nextStatus = status.StatusStarted
		event      events.Event
	)

	switch step := task.Config.Config.(type) {
	case *model.InitStep:
		event = events.InitContainerStart{
			EventOrigin: origin,
			Config: events.ContainerConfig{
				Image: step.Image,
				Cwd:   step.Cwd,
				Env:   step.Env,
			},
		}

	case *model.ScriptStep:
		event = events.ScriptStart{
			EventOrigin: origin,
			Config: events.ScriptConfig{
				ContainerID: step.ContainerID,
				Command:     step.Command,
				Args:        step.Args,
			},
		}

	case *model.CleanupStep:
		event = events.CleanupContainer{
			EventOrigin: origin,
			ContainerID: step.ContainerID,
		}

		nextStatus = status.StatusSucceeded

	default:
		log.G(ctx).Errorf("task watcher unknown task type %T", step)
		return nextStatus, nil, ErrTaskNotFound
	}

	return nextStatus, event, nil
}

func (p TaskProcessor) taskAbort(ctx context.Context, event events.TaskAbort) error {
	task, found, err := p.tasks.Task(ctx, event.Origin().ID)
	if err != nil {
		return err
	}

	if !found {
		return ErrTaskNotFound
	}

	if task.Status().IsFinished() {
		return nil
	}

	return task.WithTx(ctx, func(txCtx context.Context) error {
		err = task.Lock(txCtx)
		if err != nil {
			return err
		}

		err = p.abortStep(txCtx, task.ID(), task.Config())
		if err != nil {
			return err
		}

		err = task.Finish(ctx, status.StatusAborted)
		if err != nil {
			return err
		}

		err = p.publishStatusChanged(ctx, task.Model())
		if err != nil {
			return err
		}

		return nil
	})
}

func (p TaskProcessor) abortStep(ctx context.Context, taskID string, step model.StepConfig) (err error) {
	switch config := step.(type) {
	case *model.ScriptStep:
		err = p.publish(ctx, events.ScriptAbort{EventOrigin: events.NewEventOrigin(taskID)})
	default:
		log.G(ctx).Debugf("cannot abort task with step type %T", config)
	}

	if err != nil {
		log.G(ctx).WithError(err).Errorf("failed to abort step")
	}

	return nil
}

func (p TaskProcessor) publishStatusChanged(ctx context.Context, task model.Task) error {
	event := events.TaskStatus{
		ChangeStatus: events.ChangeStatus{
			EventOrigin: events.NewEventOrigin(task.ID),
			Status:      task.Status,
		},
		JobID: task.JobID,
	}

	return p.publish(ctx, event)
}

func (p TaskProcessor) publish(ctx context.Context, event events.Event) error {
	return p.publisher.Publish(ctx, event)
}
