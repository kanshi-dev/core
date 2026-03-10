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


CREATE TABLE agents
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