-- name: UpsertAgentHeartbeat :exec
UPDATE agents
SET last_seen = NOW()
WHERE agent_id = $1;



-- name: ListAgents :many
SELECT
    agent_id AS "agentId",
    hostname AS "hostName",
    os AS "os",
    arch AS "arch",
    cpu_cores AS "cpuCores",
    total_memory AS "totalMemory",
    version AS "version",
    last_seen AS "lastSeen"
FROM agents
ORDER BY last_seen DESC;


-- name: UpsertAgentReport :exec
INSERT INTO agents (
    agent_id,
    hostname,
    os,
    arch,
    cpu_cores,
    total_memory,
    version,
    last_seen
)
VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())
ON CONFLICT (agent_id)
    DO UPDATE SET
                  hostname = EXCLUDED.hostname,
                  os = EXCLUDED.os,
                  arch = EXCLUDED.arch,
                  cpu_cores = EXCLUDED.cpu_cores,
                  total_memory = EXCLUDED.total_memory,
                  version = EXCLUDED.version,
                  last_seen = EXCLUDED.last_seen;

