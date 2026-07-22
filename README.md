# Kanshi Core

[![CI](https://github.com/kanshi-dev/core/actions/workflows/ci.yaml/badge.svg)](https://github.com/kanshi-dev/core/actions/workflows/ci.yaml)

Kanshi Core is a gRPC and REST server that manages metrics and agent data for the Kanshi monitoring system.

## Features

- **gRPC Ingestion**: Receives `IngestBatch` and `ReportAgent` requests from Kanshi Agent.
- **Persistent Storage**: Stores metrics and agent metadata in TimescaleDB.
- **REST API**: Provides endpoints for querying agent status and historical metrics.

## Architecture

- **gRPC Server**: Port `:50051` (for agent communication)
- **REST API**: Port `:8080` (for querying data)
- **Database**: TimescaleDB (PostgreSQL with time-series extensions)

## REST API Endpoints

- `GET /api/v1/agents`: List all registered agents and their current status (online/offline).
- `GET /api/v1/metrics`: Query historical metrics for a specific agent. Explicit `from` and `to` RFC3339 timestamps may span at most one hour.
- `GET /api/v1/metrics/aggregate`: Query aggregated metrics (min, max, avg) over a time interval.
    - Example: `${API_URL}/api/v1/metrics/aggregate?agentId=${agentId}&name=${name}&interval=30s`
    - Supported Intervals: `30s`, `1m`, `5m`, `15m`
    - Supported Metrics: `cpu.used_percent`, `mem.used_percent`, `disk.used_percent`

## Run

```bash
# Ensure you have a running PostgreSQL/TimescaleDB instance
export KANSHI_API_KEY=replace-with-a-shared-secret
go run cmd/core/main.go
```

`KANSHI_API_KEY` is required and must match the value configured on every agent.

## Database Setup

Schema is located in `db/schema/schema.sql`.

## Purpose

This project is the central component of the Kanshi data pipeline: Agent → [Core](https://github.com/kanshi-dev/core)
# Quickstart and support

See the [v1 quickstart](QUICKSTART.md) to run the release stack and install an agent.

Kanshi follows semantic versioning from `v1.0.0`. Bug fixes ship in `v1.0.x`, features wait for the next minor release, and breaking API changes wait for the next major release. Release notes are generated from merged pull requests. Use GitHub issues for public support and [private vulnerability reporting](SECURITY.md) for security reports. The latest `v1.0.x` release is supported.
