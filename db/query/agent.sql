-- name: UpsertAgentHeartbeat :exec
UPDATE agents
SET last_seen = NOW()
WHERE agent_id = $1;



-- name: ListAgents :many
SELECT
    agent_id AS "agentId",
    hostname AS "hostName",
    os AS "os",
    platform,
    arch AS "arch",
    cpu_cores AS "cpuCores",
    total_memory AS "totalMemory",
    version AS "version",
    last_seen AS "lastSeen",
    disk_size AS "diskSize"
FROM agents
ORDER BY last_seen DESC;


-- name: UpsertAgentReport :exec
INSERT INTO agents (
    agent_id,
    hostname,
    os,
    platform,
    arch,
    cpu_cores,
    total_memory,
    disk_size,
    version,
    last_seen
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())
ON CONFLICT (agent_id)
    DO UPDATE SET
                  hostname = EXCLUDED.hostname,
                  os = EXCLUDED.os,
                  platform = EXCLUDED.platform,
                  disk_size = EXCLUDED.disk_size,
                  last_seen = NOW(),
                  arch = EXCLUDED.arch,
                  cpu_cores = EXCLUDED.cpu_cores,
                  total_memory = EXCLUDED.total_memory,
                  version = EXCLUDED.version;

