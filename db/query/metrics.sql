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
    time_bucket(@interval, ts) AS bucket,
    ROUND(AVG(value)::numeric, 2)::float8 AS avg_value,
    ROUND(MIN(value)::numeric, 2)::float8 AS min_value,
    ROUND(MAX(value)::numeric, 2)::float8 AS max_value,
    ROUND(
                    percentile_cont(0.95) WITHIN GROUP (ORDER BY value)::numeric,
                    2
    )::float8 AS p95_value
FROM metrics
WHERE agent_id = @agent_id
  AND name = @name
  AND ts BETWEEN @from_ts AND @to_ts
GROUP BY bucket
ORDER BY bucket;