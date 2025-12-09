-- name: TaskEvent :many
SELECT * FROM task_events
WHERE task_id = $1 AND event_type = $2
ORDER BY occured_at DESC;

-- name: TaskEvents :many
SELECT * FROM task_events
WHERE task_id = $1
ORDER BY occured_at DESC;

-- name: CreateTaskEvent :exec
INSERT INTO task_events (task_id, event_type, occured_at, payload) VALUES ($1, $2, $3, $4);
