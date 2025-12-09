-- name: Task :one
SELECT * FROM tasks
WHERE id = $1 LIMIT 1;

-- name: Tasks :many
SELECT * FROM tasks;

-- name: TasksByStatus :many
SELECT * FROM tasks
WHERE status = $1;

-- name: TasksByBuild :many
SELECT * FROM tasks
WHERE build_id = $1;

-- name: TasksByBuildAndStatus :many
SELECT * FROM tasks
WHERE build_id = $1 AND status = $2;

-- name: CreateTask :exec
INSERT INTO tasks (id, build_id, status, step) VALUES ($1, $2, $3, $4);
