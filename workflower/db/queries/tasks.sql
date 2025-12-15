-- name: Task :one
SELECT * FROM tasks
  WHERE id = $1 LIMIT 1;

-- name: LockTask :one
SELECT * FROM tasks
  WHERE id = $1 LIMIT 1
  FOR UPDATE;

-- name: Tasks :many
SELECT * FROM tasks;

-- name: TasksByStatus :many
SELECT * FROM tasks
  WHERE status = $1;

-- name: TasksByJob :many
SELECT * FROM tasks
  WHERE job_id = $1;

-- name: TasksByJobAndStatus :many
SELECT * FROM tasks
  WHERE job_id = $1 AND status = $2;

-- name: CreateTask :one
INSERT INTO tasks (id, job_id, name, status, config) 
  VALUES ($1, $2, $3, $4, $5)
  RETURNING *;

-- name: CreateEmptyTask :one
INSERT INTO tasks (id, job_id, name, status) 
  VALUES ($1, $2, $3, $4)
  RETURNING *;

-- name: UpdateTaskStatus :one
UPDATE tasks
  SET status = $2
  WHERE id = $1
  RETURNING *;

-- name: UpdateTaskConfig :one
UPDATE tasks
  SET config = $2
  WHERE id = $1
  RETURNING *;
