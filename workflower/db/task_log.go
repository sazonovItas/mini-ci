package db

import (
	"context"
	"time"

	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/workflower/db/gen/psql"
)

type TaskLogRepository struct {
	queries *Queries
}

func NewTaskLogRepository(queries *Queries) *TaskLogRepository {
	return &TaskLogRepository{queries: queries}
}

func (r *TaskLogRepository) Save(ctx context.Context, taskID string, logs ...events.LogMessage) error {
	return r.queries.WithTx(ctx, func(txCtx context.Context) error {
		queries := r.queries.Queries(txCtx)

		for _, log := range logs {
			err := queries.SaveTaskLog(
				ctx,
				psql.SaveTaskLogParams{
					TaskID:  taskID,
					Message: log.Msg,
					Time:    log.Time,
				},
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *TaskLogRepository) LastLogs(ctx context.Context, taskID string, offset, limit int) ([]events.LogMessage, error) {
	const (
		defaultLimit int = 100
	)

	queries := r.queries.Queries(ctx)

	if limit == 0 {
		limit = defaultLimit
	}

	dbLogs, err := queries.LastTaskLogsWithLimit(
		ctx,
		psql.LastTaskLogsWithLimitParams{
			TaskID: taskID,
			Limit:  int32(limit),
			Offset: int32(offset),
		},
	)
	if err != nil {
		return nil, err
	}

	logs := make([]events.LogMessage, 0, len(dbLogs))
	for _, l := range dbLogs {
		logs = append(logs, events.LogMessage{Msg: l.Message, Time: l.Time})
	}

	return logs, nil
}

func (r *TaskLogRepository) LogsSince(ctx context.Context, taskID string, since time.Time, limit int) ([]events.LogMessage, error) {
	const (
		defaultLimit int = 100
	)

	queries := r.queries.Queries(ctx)

	if limit == 0 {
		limit = defaultLimit
	}

	dbLogs, err := queries.TaskLogsSinceWithLimit(
		ctx,
		psql.TaskLogsSinceWithLimitParams{
			TaskID: taskID,
			Time:   since,
			Limit:  int32(limit),
		},
	)
	if err != nil {
		return nil, err
	}

	logs := make([]events.LogMessage, 0, len(dbLogs))
	for _, l := range dbLogs {
		logs = append(logs, events.LogMessage{Msg: l.Message, Time: l.Time})
	}

	return logs, nil
}
