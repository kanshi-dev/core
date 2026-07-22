# Kanshi quickstart

## Start the server

Install Docker with Compose, then download the release stack and create its configuration:

```sh
mkdir kanshi && cd kanshi
curl -fsSLO https://raw.githubusercontent.com/kanshi-dev/infra/main/docker-compose.yaml
curl -fsSLo .env https://raw.githubusercontent.com/kanshi-dev/infra/main/.env.example
db_password=$(openssl rand -hex 32)
ingest_key=$(openssl rand -hex 32)
dashboard_key=$(openssl rand -hex 32)
sed -i.bak "s/generate-db-password/$db_password/; s/generate-ingest-key/$ingest_key/; s/generate-dashboard-key/$dashboard_key/" .env
rm -f .env.bak
docker compose up -d
```

Core initializes the database and applies the 30-day metrics retention policy. Open `http://localhost:3000` and paste the `KANSHI_DASHBOARD_KEY` value from `.env`. REST is on port 8080 and agent gRPC ingest is on port 50051.

## Install an agent

Use the ingest key and an address reachable from the monitored host:

```sh
curl -fsSL https://kanshi.dev/install.sh | sh
export KANSHI_CORE_ADDR=your-server:50051
export KANSHI_API_KEY=the-ingest-key-from-server-env
kanshi-agent
```

On systemd Linux, install and start the service in one step:

```sh
curl -fsSL https://kanshi.dev/install.sh | sudo KANSHI_CORE_ADDR=your-server:50051 KANSHI_API_KEY=the-ingest-key-from-server-env sh -s -- --systemd
```

Set `KANSHI_VERSION=v1.0.0` to select another installer release. Set `PREFIX` to change the binary prefix.

## Verify and query

The dashboard should show the new agent after its first report. Core health is public:

```sh
curl http://localhost:8080/health
curl -H "Authorization: Bearer $dashboard_key" "http://localhost:8080/api/v1/agents"
```

Metric queries default to the latest hour. Explicit `from` and `to` values must be RFC3339 and span no more than one hour.

## Retention

Metrics are retained for 30 days. To change it, connect to the database and replace the policy:

```sql
SELECT remove_retention_policy('metrics', if_exists => TRUE);
SELECT add_retention_policy('metrics', INTERVAL '90 days');
```

## Troubleshooting

- `401 unauthorized`: use the dashboard key for REST and the ingest key for agents.
- Core health is degraded: run `docker compose logs db core` and verify `DB_PASSWORD` matches.
- Agent cannot connect: confirm port 50051 is reachable and `KANSHI_CORE_ADDR` has no URL scheme.
- Dashboard cannot load agents: confirm port 8080 is reachable through the dashboard proxy and clear then re-enter the key.
