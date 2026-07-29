-- name: InsertSpan :exec
INSERT INTO otel_spans (
    trace_id, span_id, parent_span_id, service_name, operation, span_kind, status_code,
    status_message, start_time, end_time, duration_ms, attributes
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT DO NOTHING;

-- name: InsertLog :exec
INSERT INTO otel_logs (ts, service_name, severity, body, trace_id, span_id, attributes)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListServiceSummaries :many
SELECT
    service_name,
    COUNT(*)::BIGINT AS request_count,
    COUNT(*) FILTER (WHERE status_code = 2)::BIGINT AS error_count,
    COALESCE(AVG(duration_ms), 0)::DOUBLE PRECISION AS avg_duration_ms,
    COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY duration_ms), 0)::DOUBLE PRECISION AS p95_duration_ms
FROM otel_spans
WHERE (span_kind = 2 OR parent_span_id = '')
  AND start_time >= sqlc.arg(from_time)
  AND start_time <= sqlc.arg(to_time)
GROUP BY service_name
ORDER BY service_name
LIMIT sqlc.arg(result_limit);

-- name: SearchTraces :many
SELECT
    trace_id,
    service_name,
    (ARRAY_AGG(operation ORDER BY start_time))[1]::TEXT AS root_operation,
    MIN(start_time)::TIMESTAMPTZ AS start_time,
    MAX(end_time)::TIMESTAMPTZ AS end_time,
    (EXTRACT(EPOCH FROM (MAX(end_time) - MIN(start_time))) * 1000)::DOUBLE PRECISION AS duration_ms,
    MAX(status_code)::SMALLINT AS status_code,
    COUNT(*)::BIGINT AS span_count
FROM otel_spans
WHERE start_time >= sqlc.arg(from_time)
  AND start_time <= sqlc.arg(to_time)
  AND (sqlc.arg(service_name)::TEXT = '' OR service_name = sqlc.arg(service_name))
  AND (sqlc.arg(trace_id)::TEXT = '' OR trace_id = sqlc.arg(trace_id))
GROUP BY trace_id, service_name
HAVING (sqlc.arg(status_code)::SMALLINT < 0 OR MAX(status_code) = sqlc.arg(status_code))
   AND (sqlc.arg(min_duration_ms)::DOUBLE PRECISION <= 0
        OR (EXTRACT(EPOCH FROM (MAX(end_time) - MIN(start_time))) * 1000) >= sqlc.arg(min_duration_ms))
ORDER BY MIN(start_time) DESC
LIMIT sqlc.arg(result_limit);

-- name: GetTraceSpans :many
SELECT trace_id, span_id, parent_span_id, service_name, operation, span_kind, status_code,
       status_message, start_time, end_time, duration_ms, attributes
FROM otel_spans
WHERE trace_id = $1
ORDER BY start_time, span_id
LIMIT 1000;

-- name: SearchLogs :many
SELECT ts, service_name, severity, body, trace_id, span_id, attributes
FROM otel_logs
WHERE ts >= sqlc.arg(from_time)
  AND ts <= sqlc.arg(to_time)
  AND (sqlc.arg(service_name)::TEXT = '' OR service_name = sqlc.arg(service_name))
  AND (sqlc.arg(trace_id)::TEXT = '' OR trace_id = sqlc.arg(trace_id))
  AND (sqlc.arg(span_id)::TEXT = '' OR span_id = sqlc.arg(span_id))
ORDER BY ts DESC
LIMIT sqlc.arg(result_limit);
