-- name: Workflow :one
SELECT * FROM workflows
  WHERE id = $1 LIMIT 1;

-- name: WorkflowByName :one
SELECT * FROM workflows
  WHERE name = $1 LIMIT 1;

-- name: Workflows :many
SELECT * FROM workflows 
  LIMIT $1 OFFSET $2;

-- name: CreateWorkflow :one
INSERT INTO workflows (id, name, config) 
  VALUES ($1, $2, $3)
  RETURNING *;

-- name: UpdateWorkflow :one
UPDATE workflows
  SET name = $2,
    config = $3
  WHERE id = $1
  RETURNING *;

-- name: UpdateWorkflowCurrentBuild :one
UPDATE workflows
  SET curr_build_id = $2
  WHERE id = $1
  RETURNING *;

-- name: DeleteWorkflow :exec
DELETE FROM workflows
  WHERE id = $1;
