CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS metrics (
                                       agent_id TEXT NOT NULL,
                                       name TEXT NOT NULL,
                                       value DOUBLE PRECISION NOT NULL,
                                       ts TIMESTAMPTZ NOT NULL,
                                       tags TEXT[]
);

SELECT create_hypertable('metrics', 'ts', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_metrics_name_ts
    ON metrics (name, ts DESC);

CREATE INDEX IF NOT EXISTS idx_metrics_agent_ts
    ON metrics (agent_id, ts DESC);