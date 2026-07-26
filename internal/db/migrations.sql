CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS metrics (
    agent_id TEXT NOT NULL,
    name TEXT NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    ts TIMESTAMPTZ NOT NULL,
    tags TEXT[]
);

SELECT create_hypertable('metrics', 'ts', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_metrics_name_ts ON metrics (name, ts DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_agent_ts ON metrics (agent_id, ts DESC);

CREATE TABLE IF NOT EXISTS agents (
    agent_id TEXT PRIMARY KEY,
    hostname TEXT NOT NULL,
    os TEXT NOT NULL,
    platform TEXT NOT NULL,
    arch TEXT NOT NULL,
    cpu_cores INT NOT NULL,
    total_memory BIGINT NOT NULL,
    disk_size BIGINT NOT NULL,
    version TEXT NOT NULL,
    last_seen TIMESTAMPTZ NOT NULL
);

SELECT add_retention_policy('metrics', INTERVAL '30 days', if_not_exists => TRUE);

CREATE TABLE IF NOT EXISTS alert_rules (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL,
    metric TEXT NOT NULL,
    comparator TEXT NOT NULL,
    threshold DOUBLE PRECISION NOT NULL,
    agent_id TEXT,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS alert_events (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    rule_id BIGINT NOT NULL REFERENCES alert_rules (id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL,
    state TEXT NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    webhook_status TEXT NOT NULL DEFAULT 'pending',
    webhook_error TEXT
);

CREATE INDEX IF NOT EXISTS idx_alert_events_rule_agent_ts ON alert_events (rule_id, agent_id, created_at DESC);
