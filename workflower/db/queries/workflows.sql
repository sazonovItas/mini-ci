-- name: Workflow :one
SELECT * FROM workflows
  WHERE id = $1 LIMIT 1;

-- name: WorkflowByName :one
SELECT * FROM workflows
  WHERE name = $1 LIMIT 1;

-- name: Workflows :many
SELECT * FROM workflows 
  OFFSET $1 LIMIT $2;

-- name: CreateWorkflow :one
INSERT INTO workflows (id, name, config) 
  VALUES ($1, $2, $3)
  RETURNING *;
