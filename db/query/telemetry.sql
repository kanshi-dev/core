-- name: InsertSpan :exec
INSERT INTO otel_spans (
    trace_id, span_id, parent_span_id, service_name, operation, span_kind, status_code,
    status_message, start_time, end_time, duration_ms, attributes, resource_agent_id,
    resource_host_name
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
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
    COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY duration_ms), 0)::DOUBLE PRECISION AS p95_duration_ms,
    COALESCE((
        SELECT jsonb_agg(jsonb_build_object('agentId', host.agent_id, 'hostName', host.host_name)
                         ORDER BY host.host_name, host.agent_id)
        FROM (
            SELECT DISTINCT resolved.agent_id,
                   COALESCE(identity_span.resource_host_name, resolved.hostname) AS host_name
            FROM otel_spans identity_span
            LEFT JOIN LATERAL (
                SELECT agents.agent_id, agents.hostname
                FROM agents
                WHERE agents.agent_id = identity_span.resource_agent_id
                   OR agents.hostname = identity_span.resource_host_name
                ORDER BY (agents.agent_id = identity_span.resource_agent_id) DESC, agents.last_seen DESC
                LIMIT 1
            ) resolved ON TRUE
            WHERE identity_span.service_name = summary_span.service_name
              AND identity_span.start_time >= sqlc.arg(from_time)
              AND identity_span.start_time <= sqlc.arg(to_time)
              AND (identity_span.resource_host_name IS NOT NULL OR resolved.agent_id IS NOT NULL)
            ORDER BY host_name, resolved.agent_id
            LIMIT 20
        ) host
    ), '[]'::jsonb)::TEXT AS hosts
FROM otel_spans summary_span
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
       status_message, start_time, end_time, duration_ms, attributes,
       COALESCE(resolved.agent_id, '') AS host_agent_id,
       COALESCE(otel_spans.resource_host_name, resolved.hostname, '') AS host_name
FROM otel_spans
LEFT JOIN LATERAL (
    SELECT agents.agent_id, agents.hostname
    FROM agents
    WHERE agents.agent_id = otel_spans.resource_agent_id
       OR agents.hostname = otel_spans.resource_host_name
    ORDER BY (agents.agent_id = otel_spans.resource_agent_id) DESC, agents.last_seen DESC
    LIMIT 1
) resolved ON TRUE
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
