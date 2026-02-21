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


-- name: GetMetricsByTimeRange :many
SELECT
    agent_id,
    name,
    ROUND(value::numeric, 2)::float8 AS value,
    ts,
    tags
FROM metrics
WHERE agent_id = @agent_id
  AND name = @name
  AND ts BETWEEN @from_ts AND @to_ts
ORDER BY ts DESC
LIMIT 100;


-- name: GetAggregatedMetrics :many
SELECT
    time_bucket(@interval::interval, ts) AS bucket,
    ROUND(AVG(value)) AS avg_value
FROM metrics
WHERE agent_id = @agent_id
  AND name = @name
  AND ts BETWEEN @from_ts AND @to_ts
GROUP BY bucket
ORDER BY bucket;