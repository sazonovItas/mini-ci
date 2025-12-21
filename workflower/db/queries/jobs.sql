-- name: Job :one
SELECT * FROM jobs
  WHERE id = $1;

-- name: LockJob :one
SELECT * FROM jobs
  WHERE id = $1 LIMIT 1
  FOR UPDATE;

-- name: JobsByBuild :many
SELECT * FROM jobs
  WHERE build_id = $1;

-- name: JobsByStatus :many
SELECT * FROM jobs
  WHERE status = $1;

-- name: JobsForSchedule :many
SELECT * FROM jobs
  WHERE status = $1
  ORDER BY scheduled_at
  LIMIT $2;

-- name: CreateJob :one
INSERT INTO jobs (id, build_id, name, status, config, plan)
  VALUES ($1, $2, $3, $4, $5, $6)
  RETURNING *;

-- name: CreateEmptyJob :one
INSERT INTO jobs (id, build_id, name, status)
  VALUES ($1, $2, $3, $4)
  RETURNING *;

-- name: UpdateJobStatus :one
UPDATE jobs
  SET status = $2
  WHERE id = $1
  RETURNING *;
