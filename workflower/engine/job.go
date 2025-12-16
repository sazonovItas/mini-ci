package engine

import (
	"context"
	"errors"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/core/ptr"
	"github.com/sazonovItas/mini-ci/workflower/db"
	"github.com/sazonovItas/mini-ci/workflower/model"
)

var ErrTaskNotFoundForSchedule = errors.New("task not found for schedule")

type JobProcessor struct {
	jobs      *db.JobFactory
	tasks     *db.TaskFactory
	publisher events.Publisher
}

func NewJobProcessor(
	jobs *db.JobFactory,
	tasks *db.TaskFactory,
	publisher events.Publisher,
) JobProcessor {
	return JobProcessor{jobs: jobs, tasks: tasks, publisher: publisher}
}

func (p JobProcessor) Filters() []events.FilterFunc {
	return []events.FilterFunc{
		events.WithEventTypes(
			events.EventTypeJobStatus,
			events.EventTypeTaskStatus,
		),
	}
}

func (p JobProcessor) ProcessEvent(ctx context.Context, event events.Event) (err error) {
	switch e := event.(type) {
	case events.JobStatus:
		err = p.pendingStatus(ctx, e)

	case events.TaskStatus:
		err = p.taskStatus(ctx, e)

	default:
		log.G(ctx).Errorf("job processor: unknown event type %T", e)
	}

	if err != nil {
		log.G(ctx).WithError(err).Errorf("job processor: failed to process event %T", event)
	}

	return err
}

func (p JobProcessor) pendingStatus(ctx context.Context, event events.JobStatus) error {
	if event.Status.IsPending() {
		return nil
	}

	job, found, err := p.jobs.Job(ctx, event.Origin().ID)
	if err != nil {
		return err
	}

	if !found {
		return nil
	}

	return job.WithTx(ctx, func(txCtx context.Context) error {
		err := job.Lock(txCtx)
		if err != nil {
			return err
		}

		finished, task, err := p.scheduleNextTask(txCtx, job)
		if err != nil {
			return err
		}

		if !finished {
			return nil
		}

		err = job.Finish(ctx, task.Status())
		if err != nil {
			return err
		}

		err = p.publishStatusChanged(ctx, job.Model())
		if err != nil {
			return err
		}

		return nil
	})
}

func (p JobProcessor) taskStatus(ctx context.Context, event events.TaskStatus) error {
	if event.Status.IsPending() {
		return nil
	}

	job, found, err := p.jobs.Job(ctx, event.JobID)
	if err != nil {
		return err
	}

	if !found {
		return nil
	}

	return job.WithTx(ctx, func(txCtx context.Context) error {
		err := job.Lock(txCtx)
		if err != nil {
			return err
		}

		finished, task, err := p.scheduleNextTask(txCtx, job)
		if err != nil {
			return err
		}

		if !finished {
			return nil
		}

		err = job.Finish(ctx, task.Status())
		if err != nil {
			return err
		}

		err = p.publishStatusChanged(ctx, job.Model())
		if err != nil {
			return err
		}

		return nil
	})
}

func (p JobProcessor) scheduleNextTask(ctx context.Context, job db.Job) (finished bool, task db.Task, err error) {
	var found bool

	outputs := &model.Outputs{}

	plan := ptr.To(job.Plan())
	for plan != nil {
		task, found, err = p.tasks.Task(ctx, plan.Ref.ID)
		if err != nil {
			return false, nil, err
		}

		if !found {
			return false, nil, ErrTaskNotFoundForSchedule
		}

		if task.Status().IsRunning() {
			return false, task, nil
		}

		if task.Status().IsFinished() && !task.Status().IsSucceeded() {
			return true, task, nil
		}

		if task.Status().IsSucceeded() {
			plan = plan.Next
			task.Config().Config.SetOutputs(outputs)
			continue
		}

		task.Config().Config.GetOutputs(outputs)

		if err := task.UpdateConfig(ctx, task.Config()); err != nil {
			return false, nil, err
		}

		if err := task.Pending(ctx); err != nil {
			return false, nil, err
		}

		if err := p.publishTaskStatusChanged(ctx, task.Model()); err != nil {
			return false, nil, err
		}

		return false, task, nil
	}

	return true, task, nil
}

func (p JobProcessor) publishStatusChanged(ctx context.Context, job model.Job) error {
	event := events.JobStatus{
		ChangeStatus: events.ChangeStatus{
			EventOrigin: events.NewEventOrigin(job.ID),
			Status:      job.Status,
		},
		BuildID: job.BuildID,
	}

	return p.Publish(ctx, event)
}

func (p JobProcessor) publishTaskStatusChanged(ctx context.Context, task model.Task) error {
	event := events.TaskStatus{
		ChangeStatus: events.ChangeStatus{
			EventOrigin: events.NewEventOrigin(task.ID),
			Status:      task.Status,
		},
		JobID: task.JobID,
	}

	return p.Publish(ctx, event)
}

func (p JobProcessor) Publish(ctx context.Context, event events.Event) error {
	return p.publisher.Publish(ctx, event)
}
