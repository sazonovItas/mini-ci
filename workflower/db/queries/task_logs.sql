-- name: LastTaskLogsWithLimit :many
SELECT * FROM task_logs
  WHERE task_id = $1
  ORDER BY time DESC
  LIMIT $2;

-- name: LastTaskLogsSinceWithLimit :many
SELECT * FROM task_logs
  WHERE task_id = $1 AND time > $2
  LIMIT $3;

-- name: SaveTaskLog :exec
INSERT INTO task_logs (task_id, message, time)
  VALUES ($1, $2, $3);
