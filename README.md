# Kanshi Core

[![CI](https://github.com/kanshi-dev/core/actions/workflows/ci.yaml/badge.svg)](https://github.com/kanshi-dev/core/actions/workflows/ci.yaml)

Kanshi Core receives authenticated host metrics over gRPC, stores them in TimescaleDB, and serves the authenticated REST API used by the Kanshi dashboard.

## Interfaces

| Interface | Address | Authentication |
| --- | --- | --- |
| Health | `GET :8080/health` | None |
| REST API | `/api/v1` on `:8080` | `Authorization: Bearer <KANSHI_DASHBOARD_KEY>` |
| Agent ingest | gRPC on `:50051` | `x-api-key: <KANSHI_API_KEY>` |
| OTLP traces and logs | OTLP/gRPC on `:50051` | `x-api-key: <KANSHI_API_KEY>` |

REST responses use `{ "code": 200, "message": "ok", "data": ... }`.

## REST API

- `GET /api/v1/agents`
- `GET /api/v1/metrics?agentId=&name=&from=&to=`
- `GET /api/v1/metrics/aggregate?agentId=&name=&interval=&from=&to=`
- `GET|POST /api/v1/alerts/rules`, `PUT|DELETE /api/v1/alerts/rules/:id`
- `GET /api/v1/alerts/active`
- `GET /api/v1/alerts/events?limit=`
- `GET /api/v1/services?from=&to=&limit=`
- `GET /api/v1/traces?service=&status=&minDurationMs=&traceId=&from=&to=&limit=`
- `GET /api/v1/traces/:traceId`
- `GET /api/v1/logs?service=&traceId=&spanId=&from=&to=&limit=`

Supported host metrics include CPU, memory, disk, and network send and receive rates. Aggregate intervals are `30s`, `1m`, `5m`, `15m`, and `1h`. Explicit RFC3339 metric ranges may span at most seven days, and aggregate requests may contain at most 1,000 buckets. The default range remains one hour and raw responses remain limited to 100 points.

## Application telemetry

Core accepts the standard OTLP trace and log gRPC services on port `50051`. Configure an upstream OpenTelemetry Collector exporter with the `x-api-key` header. OTLP/HTTP is not supported in v1.2.

Ingestion is bounded to a 4 MiB gRPC message, 1,000 spans or log records per request, 64 attributes per resource or record, 32 KiB of encoded attributes, and 16 KiB log bodies. Trace and span IDs, finite numeric values, timestamps, service identity, and field sizes are validated before persistence. REST searches default to one hour, allow at most 24 hours, return 100 records by default, and cap `limit` at 500.

| Variable | Default | Description |
| --- | --- | --- |
| `KANSHI_TRACE_RETENTION` | `168h` | Trace retention as a Go duration, minimum `1h` |
| `KANSHI_LOG_RETENTION` | `72h` | Correlated log retention as a Go duration, minimum `1h` |

Sample production traces. Parent-based `traceidratio` sampling at `0.1` is a reasonable starting point; adjust from measured volume. Avoid unsampled high-volume debug logs, redact secrets before export, and never attach credentials or request bodies as attributes.

## Alerting

Core evaluates persisted alert rules on a fixed schedule and records firing and resolved transitions. A rule targets `cpu.used_percent`, `mem.used_percent`, `disk.used_percent`, or `agent.offline`, globally or for a single agent, and stays disabled until enabled. State lives in the `alert_events` table and survives a restart, so a sustained breach fires once and recovery resolves once. Each transition is delivered to every configured webhook with an optional HMAC-SHA256 signature and bounded retries.

| Variable | Default | Description |
| --- | --- | --- |
| `KANSHI_ALERT_INTERVAL` | `30s` | How often rules are evaluated (Go duration) |
| `KANSHI_WEBHOOK_URLS` | *(empty)* | Comma-separated webhook URLs for alert delivery |
| `KANSHI_WEBHOOK_SECRET` | *(empty)* | Optional secret for HMAC-SHA256 payload signing |

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

Use the [local demo](https://github.com/kanshi-dev/demo) or follow [QUICKSTART.md](QUICKSTART.md). The AWS test deployment lives in [kanshi-dev/infra](https://github.com/kanshi-dev/infra).

## Support and security

Use GitHub issues for public support. Report vulnerabilities through [private vulnerability reporting](SECURITY.md). Kanshi follows semantic versioning from `v1.0.0`.
