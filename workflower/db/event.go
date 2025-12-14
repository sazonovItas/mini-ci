package db

import (
	"context"
	"encoding/json"

	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/workflower/db/gen/psql"
)

type EventRepository struct {
	queries *Queries
}

func NewEventRepository(queries *Queries) *EventRepository {
	return &EventRepository{queries: queries}
}

func (r *EventRepository) Save(ctx context.Context, event events.Event) error {
	queries := r.queries.Queries(ctx)

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	err = queries.SaveEvent(
		ctx,
		psql.SaveEventParams{
			OriginID:  event.Origin().ID,
			OccuredAt: event.Origin().OccuredAt,
			EventType: event.Type().String(),
			Payload:   payload,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *EventRepository) Events(ctx context.Context, originID string) ([]events.Event, error) {
	queries := r.queries.Queries(ctx)

	dbEvents, err := queries.Events(ctx, originID)
	if err != nil {
		return nil, err
	}

	out := make([]events.Event, 0, len(dbEvents))
	for _, e := range dbEvents {
		var event events.Event
		if err := json.Unmarshal(e.Payload, &event); err != nil {
			return nil, err
		}

		out = append(out, event)
	}

	return out, nil
}

func (r *EventRepository) EventsByType(ctx context.Context, originID string, eventType string) ([]events.Event, error) {
	queries := r.queries.Queries(ctx)

	dbEvents, err := queries.EventsByType(
		ctx,
		psql.EventsByTypeParams{
			OriginID:  originID,
			EventType: eventType,
		},
	)
	if err != nil {
		return nil, err
	}

	out := make([]events.Event, 0, len(dbEvents))
	for _, e := range dbEvents {
		var event events.Event
		if err := json.Unmarshal(e.Payload, &event); err != nil {
			return nil, err
		}

		out = append(out, event)
	}

	return out, nil
}
