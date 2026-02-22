-- name: UpsertAgentHeartbeat :exec
INSERT INTO agents (agent_id, last_seen)
VALUES ($1, NOW())
ON CONFLICT (agent_id)
    DO UPDATE SET last_seen = EXCLUDED.last_seen;


-- name: ListAgents :many
SELECT
    agent_id,
    last_seen
FROM agents
ORDER BY last_seen DESC;