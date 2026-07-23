# Kanshi Core

[![CI](https://github.com/kanshi-dev/core/actions/workflows/ci.yaml/badge.svg)](https://github.com/kanshi-dev/core/actions/workflows/ci.yaml)

Kanshi Core receives authenticated host metrics over gRPC, stores them in TimescaleDB, and serves the authenticated REST API used by the Kanshi dashboard.

## Interfaces

| Interface | Address | Authentication |
| --- | --- | --- |
| Health | `GET :8080/health` | None |
| REST API | `/api/v1` on `:8080` | `Authorization: Bearer <KANSHI_DASHBOARD_KEY>` |
| Agent ingest | gRPC on `:50051` | `x-api-key: <KANSHI_API_KEY>` |

REST responses use `{ "code": 200, "message": "ok", "data": ... }`.

## REST API

- `GET /api/v1/agents`
- `GET /api/v1/metrics?agentId=&name=&from=&to=`
- `GET /api/v1/metrics/aggregate?agentId=&name=&interval=`

Supported metrics are `cpu.used_percent`, `mem.used_percent`, and `disk.used_percent`. Aggregate intervals are `30s`, `1m`, `5m`, and `15m`. Explicit RFC3339 metric ranges may span at most one hour.

## Run from source

Start TimescaleDB, then configure both shared keys:

```sh
export DB_HOST=127.0.0.1
export DB_PASSWORD=replace-me
export KANSHI_API_KEY=replace-with-an-ingest-key
export KANSHI_DASHBOARD_KEY=replace-with-a-dashboard-key
go run ./cmd/core
```

Core applies its schema and 30-day retention policy at startup. SQL lives in `db/schema` and `db/query`; regenerate sqlc output with `sqlc generate`. Never hand-edit `internal/db/*.sql.go`.

## Verify

```sh
go test ./...
go vet ./...
go build ./...
```

## Start the complete stack

Use the [local demo](https://github.com/kanshi-dev/demo) or follow [QUICKSTART.md](QUICKSTART.md). The AWS test deployment lives in [kanshi-dev/infra](https://github.com/kanshi-dev/infra/tree/main/deployment/infra).

## Support and security

Use GitHub issues for public support. Report vulnerabilities through [private vulnerability reporting](SECURITY.md). Kanshi follows semantic versioning from `v1.0.0`.
