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

-- name: CreateJob :exec
INSERT INTO jobs (id, build_id, name, status, plan)
  VALUES ($1, $2, $3, $4, $5);

-- name: UpdateJobStatus :exec
UPDATE jobs
  SET status = $2
  WHERE id = $1;
