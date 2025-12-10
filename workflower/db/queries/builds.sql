-- name: Build :one
SELECT * FROM builds
  WHERE id = $1 LIMIT 1;

-- name: BuildsByWorkflow :many
SELECT * FROM builds
  WHERE workflow_id = $1;

-- name: CreateBuild :exec
INSERT INTO builds (id, workflow_id, status, plan) VALUES ($1, $2, $3, $4);

-- name: UpdateBuildStatus :exec
UPDATE builds
  SET status = $2
  WHERE id = $1;
