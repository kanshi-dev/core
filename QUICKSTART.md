# Kanshi quickstart

The current stable release uses Core, Agent, and Dashboard `v1.0.0`.

## Start locally

```sh
git clone https://github.com/kanshi-dev/demo.git
cd demo
cp .env.example .env

db_password=$(openssl rand -hex 32)
ingest_key=$(openssl rand -hex 32)
dashboard_key=$(openssl rand -hex 32)
sed -i.bak "s/generate-db-password/$db_password/; s/generate-ingest-key/$ingest_key/; s/generate-dashboard-key/$dashboard_key/" .env
rm -f .env.bak

docker compose up -d
```

Core initializes TimescaleDB and applies the 30-day retention policy. Open `http://localhost:3000` and enter `KANSHI_DASHBOARD_KEY` from `.env`.

## Install an agent

```sh
curl -fsSL https://kanshi.dev/install.sh |
  KANSHI_VERSION=v1.0.0 sh

export KANSHI_CORE_ADDR=your-server:50051
export KANSHI_API_KEY=the-ingest-key-from-.env
kanshi-agent
```

For systemd Linux:

```sh
curl -fsSL https://kanshi.dev/install.sh |
  sudo KANSHI_VERSION=v1.0.0 \
  KANSHI_CORE_ADDR=your-server:50051 \
  KANSHI_API_KEY=the-ingest-key-from-.env \
  sh -s -- --systemd
```

## Verify

```sh
curl http://localhost:8080/health
curl -H "Authorization: Bearer $dashboard_key" \
  http://localhost:8080/api/v1/agents
```

Metric queries default to the latest hour. Explicit `from` and `to` values must be RFC3339 and span no more than one hour.

## Troubleshooting

- `401 unauthorized`: use the dashboard key for REST and the ingest key for agents.
- Degraded health: run `docker compose logs db core`.
- Agent connection failure: confirm `50051` is reachable and omit URL schemes from `KANSHI_CORE_ADDR`.
- Dashboard load failure: clear the stored key and enter `KANSHI_DASHBOARD_KEY` again.

Use `docker compose down` to stop the stack. Add `-v` only when you also want to delete local metrics.
