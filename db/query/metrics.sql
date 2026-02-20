-- name: InsertMetricsBatch :exec
INSERT INTO metrics (
    agent_id,
    name,
    value,
    ts,
    tags
)
SELECT
    unnest(@agent_id_s::text[]),
    unnest(@names::text[]),
    unnest(@values::double precision[]),
    unnest(@timestamps::timestamptz[]),
    @tags::text[][];