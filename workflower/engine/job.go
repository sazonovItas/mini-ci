package engine

import (
	"context"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/core/status"
	"github.com/sazonovItas/mini-ci/workflower/db"
	"github.com/sazonovItas/mini-ci/workflower/model"
)

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
			events.EventTypeJobAbort,
		),
	}
}

func (p JobProcessor) ProcessEvent(ctx context.Context, event events.Event) (err error) {
	switch e := event.(type) {
	case events.JobStatus:
		err = p.jobStatus(ctx, e)

	case events.TaskStatus:
		err = p.taskStatus(ctx, e)

	case events.JobAbort:
		err = p.jobAbort(ctx, e)

	default:
		log.G(ctx).Errorf("job processor: unknown event type %T", e)
	}

	if err != nil {
		log.G(ctx).WithError(err).Errorf("job processor: failed to process event %T", event)
	}

	return err
}

func (p JobProcessor) jobStatus(ctx context.Context, event events.JobStatus) error {
	if !event.Status.IsPending() {
		return nil
	}

	job, found, err := p.jobs.Job(ctx, event.Origin().ID)
	if err != nil {
		return err
	}

	if !found {
		return ErrJobNotFound
	}

	if job.Status().IsStarted() {
		return nil
	}

	// return job.WithTx(ctx, func(txCtx context.Context) error {
	// err := job.Lock(txCtx)
	// if err != nil {
	// 	return err
	// }

	outputs := &model.Outputs{}

	_, err = p.scheduleNextTask(ctx, outputs, job.Plan())
	if err != nil {
		return err
	}

	err = job.Start(ctx)
	if err != nil {
		return err
	}

	return nil
	// })
}

func (p JobProcessor) taskStatus(ctx context.Context, event events.TaskStatus) error {
	if !event.Status.IsFinished() {
		return nil
	}

	job, found, err := p.jobs.Job(ctx, event.JobID)
	if err != nil {
		return err
	}

	if !found {
		return ErrJobNotFound
	}

	if job.Status().IsFinished() {
		return nil
	}

	// return job.WithTx(ctx, func(txCtx context.Context) error {
	// err := job.Lock(txCtx)
	// if err != nil {
	// 	return err
	// }

	outputs := &model.Outputs{}

	taskStatus, err := p.scheduleNextTask(ctx, outputs, job.Plan())
	if err != nil {
		return err
	}

	if taskStatus.IsFinished() {
		err = job.Finish(ctx, taskStatus)
		if err != nil {
			return err
		}

		err = p.publishStatusChanged(ctx, job.Model())
		if err != nil {
			return err
		}
	}

	return nil
	// })
}

func (p JobProcessor) scheduleNextTask(ctx context.Context, outputs *model.Outputs, plan model.TaskPlan) (status.Status, error) {
	task, found, err := p.tasks.Task(ctx, plan.Ref.ID)
	if err != nil {
		return status.StatusUnknown, err
	}

	if !found {
		return status.StatusUnknown, ErrTaskNotFound
	}

	if task.Status().IsCreated() {
		// err = task.Lock(ctx)
		// if err != nil {
		// 	return status.StatusUnknown, err
		// }

		config := task.Config()
		config.GetOutputs(outputs)

		err = task.UpdateConfig(ctx, config)
		if err != nil {
			return status.StatusUnknown, err
		}

		err = task.Pending(ctx)
		if err != nil {
			return status.StatusUnknown, err
		}

		err = p.publishTaskStatusChanged(ctx, task.Model())
		if err != nil {
			return status.StatusUnknown, err
		}

		return task.Status(), nil
	}

	if task.Status().IsRunning() {
		return task.Status(), nil
	}

	if task.Status().IsFinished() {
		if !task.Status().IsSucceeded() {
			return task.Status(), nil
		}

		task.Config().SetOutputs(outputs)
	}

	next := plan.Next
	if next == nil {
		return task.Status(), nil
	}

	return p.scheduleNextTask(ctx, outputs, *next)
}

func (p JobProcessor) jobAbort(ctx context.Context, event events.JobAbort) error {
	job, found, err := p.jobs.Job(ctx, event.Origin().ID)
	if err != nil {
		return err
	}

	if !found {
		return ErrJobNotFound
	}

	if job.Status().IsFinished() {
		return nil
	}

	return p.jobs.WithTx(ctx, func(txCtx context.Context) error {
		err = job.Lock(txCtx)
		if err != nil {
			return err
		}

		tasks, err := job.Tasks(txCtx)
		if err != nil {
			return err
		}

		for _, task := range tasks {
			status := task.Status()

			switch {
			case status.IsFinished():
				continue

			case status.IsRunning():
				err = p.publish(txCtx, events.TaskAbort{EventOrigin: events.NewEventOrigin(task.ID())})
				if err != nil {
					return err
				}
			}
		}

		err = job.Finish(txCtx, status.StatusAborted)
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

func (p JobProcessor) publishStatusChanged(ctx context.Context, job model.Job) error {
	event := events.JobStatus{
		ChangeStatus: events.ChangeStatus{
			EventOrigin: events.NewEventOrigin(job.ID),
			Status:      job.Status,
		},
		BuildID: job.BuildID,
	}

	return p.publish(ctx, event)
}

func (p JobProcessor) publishTaskStatusChanged(ctx context.Context, task model.Task) error {
	event := events.TaskStatus{
		ChangeStatus: events.ChangeStatus{
			EventOrigin: events.NewEventOrigin(task.ID),
			Status:      task.Status,
		},
		JobID: task.JobID,
	}

	return p.publish(ctx, event)
}

func (p JobProcessor) publish(ctx context.Context, event events.Event) error {
	return p.publisher.Publish(ctx, event)
}
