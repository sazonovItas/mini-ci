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

-- name: CreateTask :exec
INSERT INTO tasks (id, job_id, name, status, config) 
  VALUES ($1, $2, $3, $4, $5);

-- name: UpdateTaskStatus :exec
UPDATE tasks
  SET status = $2
  WHERE id = $1;
