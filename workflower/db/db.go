package db

import "github.com/jackc/pgx/v5/pgxpool"

type DB struct {
	workflows *WorkflowFactory
	builds    *BuildFactory
	jobs      *JobFactory
	tasks     *TaskFactory

	event    *EventRepository
	taskLog  *TaskLogRepository
	workflow *WorkflowRepository
}

func New(pool *pgxpool.Pool) *DB {
	queries := NewQueries(pool)

	return &DB{
		workflows: NewWorkflowFactory(queries),
		builds:    NewBuildFactory(queries),
		jobs:      NewJobFactory(queries),
		tasks:     NewTaskFactory(queries),
		event:     NewEventRepository(queries),
		taskLog:   NewTaskLogRepository(queries),
		workflow:  NewWorkflowRepository(queries),
	}
}

func (db *DB) WorkflowFactory() *WorkflowFactory {
	return db.workflows
}

func (db *DB) BuildFactory() *BuildFactory {
	return db.builds
}

func (db *DB) JobFactory() *JobFactory {
	return db.jobs
}

func (db *DB) TaskFactory() *TaskFactory {
	return db.tasks
}

func (db *DB) EventRepository() *EventRepository {
	return db.event
}

func (db *DB) TaskLogRepository() *TaskLogRepository {
	return db.taskLog
}

func (db *DB) WorkflowRepository() *WorkflowRepository {
	return db.workflow
}
