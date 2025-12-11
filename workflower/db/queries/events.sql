-- name: Events :many
SELECT * FROM events
  WHERE origin_id = $1
  ORDER BY occured_at DESC;

-- name: EventsByType :many
SELECT * FROM events
  WHERE origin_id = $1 AND event_type = $2
  ORDER BY occured_at DESC;

-- name: SaveEvent :exec
INSERT INTO events (origin_id, event_type, occured_at, payload) 
  VALUES ($1, $2, $3, $4);
