-- name: Build :one
SELECT * FROM builds
  WHERE id = $1 LIMIT 1;

-- name: LockBuild :one
SELECT * FROM builds
  WHERE id = $1 LIMIT 1
  FOR UPDATE;

-- name: BuildsByWorkflow :many
SELECT * FROM builds
  WHERE workflow_id = $1
  ORDER BY created_at DESC;

-- name: BuildsByStatus :many
SELECT * FROM builds
  WHERE status = $1;

-- name: CreateBuild :one
INSERT INTO builds (id, workflow_id, status, config, plan) 
  VALUES ($1, $2, $3, $4, $5)
  RETURNING *;

-- name: CreateEmptyBuild :one
INSERT INTO builds (id, workflow_id, status) 
  VALUES ($1, $2, $3)
  RETURNING *;

-- name: UpdateBuild :one
UPDATE builds
  set status = $2,
    plan = $3
  WHERE id = $1
  RETURNING *;

-- name: UpdateBuildStatus :one
UPDATE builds
  SET status = $2
  WHERE id = $1
  RETURNING *;
