-- name: UpsertAgentHeartbeat :exec
INSERT INTO agents (
    agent_id,
    hostname,
    os,
    arch,
    version,
    last_seen
)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (agent_id)
    DO UPDATE SET hostname  = EXCLUDED.hostname,
                  os        = EXCLUDED.os,
                  arch      = EXCLUDED.arch,
                  version   = EXCLUDED.version,
                  last_seen = EXCLUDED.last_seen;


-- name: ListAgents :many
SELECT agent_id,
       last_seen
FROM agents
ORDER BY last_seen DESC;