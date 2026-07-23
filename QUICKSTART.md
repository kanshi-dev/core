# Kanshi quickstart

The current stable release uses Core, Agent, and Dashboard `v1.0.0`.

## Start locally

```sh
git clone https://github.com/kanshi-dev/demo.git
cd demo
make up
```

`make up` generates a private `.env`, pulls the stable images, starts the stack, and prints the dashboard key. Core initializes TimescaleDB and applies the 30-day retention policy. Open `http://localhost:3000` and enter the printed key. Run `make keys` to print it again.

## Install an agent

```sh
curl -fsSL https://kanshi.dev/install.sh |
  KANSHI_VERSION=v1.0.0 sh

eval "$(KANSHI_CORE_ADDR=your-server:50051 make agent-env)"
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
dashboard_key=$(sed -n 's/^KANSHI_DASHBOARD_KEY=//p' .env)
curl -H "Authorization: Bearer $dashboard_key" \
  http://localhost:8080/api/v1/agents
```

Metric queries default to the latest hour. Explicit `from` and `to` values must be RFC3339 and span no more than one hour.

## Troubleshooting

- `401 unauthorized`: use the dashboard key for REST and the ingest key for agents.
- Degraded health: run `make logs`.
- Agent connection failure: confirm `50051` is reachable and omit URL schemes from `KANSHI_CORE_ADDR`.
- Dashboard load failure: clear the stored key and enter `KANSHI_DASHBOARD_KEY` again.

Use `make down` to stop the stack. Run `make reset` to delete local metrics and regenerate keys on the next start.
