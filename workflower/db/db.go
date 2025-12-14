package db

import "github.com/jackc/pgx/v5/pgxpool"

type DB struct {
	workflow *WorkflowFactory
	build    *BuildFactory
	job      *JobFactory
	task     *TaskFactory

	event   *EventRepository
	taskLog *TaskLogRepository
}

func New(pool *pgxpool.Pool) *DB {
	queries := NewQueries(pool)

	return &DB{
		workflow: NewWorkflowFactory(queries),
		build:    NewBuildFactory(queries),
		job:      NewJobFactory(queries),
		task:     NewTaskFactory(queries),
		event:    NewEventRepository(queries),
		taskLog:  NewTaskLogRepository(queries),
	}
}

func (db *DB) WorkflowFactory() *WorkflowFactory {
	return db.workflow
}

func (db *DB) BuildFactory() *BuildFactory {
	return db.build
}

func (db *DB) JobFactory() *JobFactory {
	return db.job
}

func (db *DB) TaskFactory() *TaskFactory {
	return db.task
}

func (db *DB) EventRepository() *EventRepository {
	return db.event
}

func (db *DB) TaskLogRepository() *TaskLogRepository {
	return db.taskLog
}
