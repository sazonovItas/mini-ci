-- name: Workflow :one
SELECT * FROM workflows
  WHERE id = $1 LIMIT 1;

-- name: WorkflowByName :one
SELECT * FROM workflows
  WHERE name = $1 LIMIT 1;

-- name: CreateWorkflow :exec
INSERT INTO workflows (id, name, config) 
  VALUES ($1, $2, $3);
