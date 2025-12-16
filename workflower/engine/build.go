package engine

import (
	"context"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
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
		),
	}
}

func (p BuildProcessor) ProcessEvent(ctx context.Context, event events.Event) (err error) {
	switch e := event.(type) {
	case events.BuildStatus:
		err = p.buildStatus(ctx, e)

	case events.JobStatus:
		err = p.jobStatus(ctx, e)

	default:
		log.G(ctx).Errorf("build: processor: unknown event %T", e)
	}

	if err != nil {
		log.G(ctx).WithError(err).Errorf("build processor: failed to process event %T", event)
	}

	return err
}

func (p BuildProcessor) buildStatus(ctx context.Context, event events.BuildStatus) error {
	return nil
}

func (p BuildProcessor) jobStatus(ctx context.Context, event events.JobStatus) error {
	return nil
}

func (p BuildProcessor) publishStatusChanged(ctx context.Context, build model.Build) error {
	event := events.BuildStatus{
		EventOrigin: events.NewEventOrigin(build.ID),
		Status:      build.Status,
	}

	return p.Publish(ctx, event)
}

func (p BuildProcessor) publishJobStatusChanged(ctx context.Context, job model.Job) error {
	event := events.JobStatus{
		ChangeStatus: events.ChangeStatus{
			EventOrigin: events.NewEventOrigin(job.ID),
			Status:      job.Status,
		},
		BuildID: job.BuildID,
	}

	return p.Publish(ctx, event)
}

func (p BuildProcessor) Publish(ctx context.Context, event events.Event) error {
	return p.Publish(ctx, event)
}
