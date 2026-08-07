-- name: InsertMetricsBatch :exec
INSERT INTO metrics (
    agent_id,
    name,
    value,
    ts,
    tags
)
SELECT
    metric.agent_id,
    metric.name,
    metric.value,
    metric.ts,
    ARRAY(SELECT jsonb_array_elements_text(tag_set.value))
FROM ROWS FROM (
    unnest(@agent_id_s::text[]),
    unnest(@names::text[]),
    unnest(@values::double precision[]),
    unnest(@timestamps::timestamptz[])
) WITH ORDINALITY AS metric(agent_id, name, value, ts, position)
JOIN jsonb_array_elements(@tags::jsonb) WITH ORDINALITY AS tag_set(value, position)
USING (position);


-- name: GetMetricsByTimeRange :many
SELECT
    agent_id AS "agentId",
    name,
    ROUND(value::numeric, 2)::float8 AS value,
    ts AS "timestamp",
    tags
FROM metrics
WHERE agent_id = @agent_id
  AND name = @name
  AND ts BETWEEN @from_ts AND @to_ts
ORDER BY ts DESC
LIMIT 100;


-- name: GetAggregatedMetrics :many
SELECT
    time_bucket(@interval, ts) AS "bucket",
    ROUND(AVG(value)::numeric, 2)::float8 AS "avgValue",
    ROUND(MIN(value)::numeric, 2)::float8 AS "minValue",
    ROUND(MAX(value)::numeric, 2)::float8 AS "maxValue",
    ROUND(percentile_cont(0.95) WITHIN GROUP (ORDER BY value)::numeric,2)::float8 AS "p95Value"
FROM metrics
WHERE agent_id = @agent_id
  AND name = @name
  AND ts BETWEEN @from_ts AND @to_ts
GROUP BY bucket
ORDER BY bucket;
