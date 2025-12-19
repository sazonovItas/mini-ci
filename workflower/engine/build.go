package engine

import (
	"context"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/core/status"
	"github.com/sazonovItas/mini-ci/workflower/db"
	"github.com/sazonovItas/mini-ci/workflower/model"
)

type BuildProcessor struct {
	builds    *db.BuildFactory
	jobs      *db.JobFactory
	publisher events.Publisher
}

func NewBuildProcessor(
	builds *db.BuildFactory,
	jobs *db.JobFactory,
	publisher events.Publisher,
) BuildProcessor {
	return BuildProcessor{builds: builds, jobs: jobs, publisher: publisher}
}

func (p BuildProcessor) Filters() []events.FilterFunc {
	return []events.FilterFunc{
		events.WithEventTypes(
			events.EventTypeBuildStatus,
			events.EventTypeJobStatus,
			events.EventTypeBuildAbort,
		),
	}
}

func (p BuildProcessor) ProcessEvent(ctx context.Context, event events.Event) (err error) {
	switch e := event.(type) {
	case events.BuildStatus:
		err = p.buildStatus(ctx, e)

	case events.JobStatus:
		err = p.jobStatus(ctx, e)

	case events.BuildAbort:
		err = p.buildAbort(ctx, e)

	default:
		log.G(ctx).Errorf("build: processor: unknown event %T", e)
	}

	if err != nil {
		log.G(ctx).WithError(err).Errorf("build processor: failed to process event %T", event)
	}

	return err
}

func (p BuildProcessor) buildStatus(ctx context.Context, event events.BuildStatus) error {
	if !event.Status.IsPending() {
		return nil
	}

	build, found, err := p.builds.Build(ctx, event.Origin().ID)
	if err != nil {
		return err
	}

	if !found {
		return ErrBuildNotFound
	}

	if build.Status().IsRunning() {
		return nil
	}

	return build.WithTx(ctx, func(txCtx context.Context) error {
		err := build.Lock(txCtx)
		if err != nil {
			return err
		}

		_, err = p.scheduleNextJob(txCtx, build.Plan())
		if err != nil {
			return err
		}

		err = build.Start(txCtx)
		if err != nil {
			return err
		}

		return nil
	})
}

func (p BuildProcessor) jobStatus(ctx context.Context, event events.JobStatus) error {
	if !event.Status.IsFinished() {
		return nil
	}

	build, found, err := p.builds.Build(ctx, event.BuildID)
	if err != nil {
		return err
	}

	if !found {
		return ErrBuildNotFound
	}

	if build.Status().IsFinished() {
		return nil
	}

	return build.WithTx(ctx, func(txCtx context.Context) error {
		err := build.Lock(txCtx)
		if err != nil {
			return err
		}

		jobStatus, err := p.scheduleNextJob(txCtx, build.Plan())
		if err != nil {
			return err
		}

		if jobStatus.IsFinished() {
			err = build.Finish(txCtx, jobStatus)
			if err != nil {
				return err
			}

			err = p.publishStatusChanged(ctx, build.Model())
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (p BuildProcessor) scheduleNextJob(ctx context.Context, plan model.JobPlan) (status.Status, error) {
	job, found, err := p.jobs.Job(ctx, plan.Ref.ID)
	if err != nil {
		return status.StatusUnknown, err
	}

	if !found {
		return status.StatusUnknown, ErrJobNotFound
	}

	if job.Status().IsCreated() {
		err = job.Lock(ctx)
		if err != nil {
			return status.StatusUnknown, err
		}

		err = job.Pending(ctx)
		if err != nil {
			return status.StatusUnknown, err
		}

		err = p.publishJobStatusChanged(ctx, job.Model())
		if err != nil {
			return status.StatusUnknown, err
		}

		return job.Status(), err
	}

	if job.Status().IsRunning() {
		return job.Status(), nil
	}

	if !job.Status().IsSucceeded() {
		return job.Status(), nil
	}

	next := plan.Next
	if next == nil {
		return job.Status(), nil
	}

	return p.scheduleNextJob(ctx, *next)
}

func (p BuildProcessor) buildAbort(ctx context.Context, event events.BuildAbort) error {
	build, found, err := p.builds.Build(ctx, event.Origin().ID)
	if err != nil {
		return err
	}

	if !found {
		return ErrBuildNotFound
	}

	if build.Status().IsFinished() {
		return nil
	}

	return p.builds.WithTx(ctx, func(txCtx context.Context) error {
		err = build.Lock(txCtx)
		if err != nil {
			return err
		}

		jobs, err := build.Jobs(txCtx)
		if err != nil {
			return err
		}

		for _, job := range jobs {
			status := job.Status()

			switch {
			case status.IsFinished():
				continue

			case status.IsRunning():
				err = p.publish(txCtx, events.JobAbort{EventOrigin: events.NewEventOrigin(job.ID())})
				if err != nil {
					return err
				}
			}
		}

		err = build.Finish(txCtx, status.StatusAborted)
		if err != nil {
			return err
		}

		err = p.publishStatusChanged(txCtx, build.Model())
		if err != nil {
			return err
		}

		return nil
	})
}

func (p BuildProcessor) publishStatusChanged(ctx context.Context, build model.Build) error {
	event := events.BuildStatus{
		EventOrigin: events.NewEventOrigin(build.ID),
		Status:      build.Status,
	}

	return p.publish(ctx, event)
}

func (p BuildProcessor) publishJobStatusChanged(ctx context.Context, job model.Job) error {
	event := events.JobStatus{
		ChangeStatus: events.ChangeStatus{
			EventOrigin: events.NewEventOrigin(job.ID),
			Status:      job.Status,
		},
		BuildID: job.BuildID,
	}

	return p.publish(ctx, event)
}

func (p BuildProcessor) publish(ctx context.Context, event events.Event) error {
	return p.publisher.Publish(ctx, event)
}
