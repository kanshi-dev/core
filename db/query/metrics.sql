-- name: InsertMetric :exec
INSERT INTO metrics (
    agent_id,
    name,
    value,
    ts,
    tags
)
VALUES ($1, $2, $3, $4, $5);