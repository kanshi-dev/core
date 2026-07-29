CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS metrics
(
    agent_id TEXT             NOT NULL,
    name     TEXT             NOT NULL,
    value    DOUBLE PRECISION NOT NULL,
    ts       TIMESTAMPTZ      NOT NULL,
    tags     TEXT[]
);

SELECT create_hypertable('metrics', 'ts', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_metrics_name_ts
    ON metrics (name, ts DESC);

CREATE INDEX IF NOT EXISTS idx_metrics_agent_ts
    ON metrics (agent_id, ts DESC);


CREATE TABLE IF NOT EXISTS agents
(
    agent_id     TEXT PRIMARY KEY,
    hostname     TEXT        NOT NULL,
    os           TEXT        NOT NULL,
    platform    TEXT        NOT NULL,
    arch         TEXT        NOT NULL,
    cpu_cores    INT         NOT NULL,
    total_memory BIGINT      NOT NULL,
    disk_size  BIGINT      NOT NULL,
    version      TEXT        NOT NULL,
    last_seen    TIMESTAMPTZ NOT NULL
);

SELECT add_retention_policy('metrics', INTERVAL '30 days', if_not_exists => TRUE);


CREATE TABLE IF NOT EXISTS alert_rules
(
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       TEXT             NOT NULL,
    metric     TEXT             NOT NULL, -- cpu.used_percent | mem.used_percent | disk.used_percent | agent.offline
    comparator TEXT             NOT NULL, -- gt | lt (agent.offline uses gt on seconds offline)
    threshold  DOUBLE PRECISION NOT NULL, -- percent, or seconds for agent.offline
    agent_id   TEXT,                      -- NULL = global (all agents)
    enabled    BOOLEAN          NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);


CREATE TABLE IF NOT EXISTS alert_events
(
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    rule_id        BIGINT           NOT NULL REFERENCES alert_rules (id) ON DELETE CASCADE,
    agent_id       TEXT             NOT NULL,
    state          TEXT             NOT NULL, -- firing | resolved
    value          DOUBLE PRECISION NOT NULL,
    created_at     TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    webhook_status TEXT             NOT NULL DEFAULT 'pending', -- pending | delivered | failed | skipped
    webhook_error  TEXT
);

CREATE INDEX IF NOT EXISTS idx_alert_events_rule_agent_ts
    ON alert_events (rule_id, agent_id, created_at DESC);

CREATE TABLE IF NOT EXISTS otel_spans
(
    trace_id TEXT NOT NULL,
    span_id TEXT NOT NULL,
    parent_span_id TEXT NOT NULL,
    service_name TEXT NOT NULL,
    operation TEXT NOT NULL,
    span_kind SMALLINT NOT NULL,
    status_code SMALLINT NOT NULL,
    status_message TEXT NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    duration_ms DOUBLE PRECISION NOT NULL,
    attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    PRIMARY KEY (trace_id, span_id, start_time)
);

SELECT create_hypertable('otel_spans', 'start_time', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_otel_spans_service_time ON otel_spans (service_name, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_otel_spans_trace ON otel_spans (trace_id, start_time);

CREATE TABLE IF NOT EXISTS otel_logs
(
    ts TIMESTAMPTZ NOT NULL,
    service_name TEXT NOT NULL,
    severity TEXT NOT NULL,
    body TEXT NOT NULL,
    trace_id TEXT NOT NULL,
    span_id TEXT NOT NULL,
    attributes JSONB NOT NULL DEFAULT '{}'::jsonb
);

SELECT create_hypertable('otel_logs', 'ts', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_otel_logs_service_time ON otel_logs (service_name, ts DESC);
CREATE INDEX IF NOT EXISTS idx_otel_logs_trace_span ON otel_logs (trace_id, span_id, ts);
