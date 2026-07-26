-- name: CreateAlertRule :one
INSERT INTO alert_rules (name, metric, comparator, threshold, agent_id, enabled)
VALUES (@name, @metric, @comparator, @threshold, @agent_id, @enabled)
RETURNING *;


-- name: GetAlertRule :one
SELECT *
FROM alert_rules
WHERE id = @id;


-- name: ListAlertRules :many
SELECT *
FROM alert_rules
ORDER BY created_at DESC;


-- name: ListEnabledAlertRules :many
SELECT *
FROM alert_rules
WHERE enabled = TRUE;


-- name: UpdateAlertRule :one
UPDATE alert_rules
SET name       = @name,
    metric     = @metric,
    comparator = @comparator,
    threshold  = @threshold,
    agent_id   = @agent_id,
    enabled    = @enabled,
    updated_at = NOW()
WHERE id = @id
RETURNING *;


-- name: DeleteAlertRule :execrows
DELETE
FROM alert_rules
WHERE id = @id;


-- Latest value per agent for one metric within a freshness window, so the
-- evaluator never fires on stale data.
-- name: GetLatestMetricPerAgent :many
SELECT DISTINCT ON (agent_id) agent_id AS "agentId",
                              value
FROM metrics
WHERE name = @name
  AND ts > NOW() - INTERVAL '5 minutes'
ORDER BY agent_id, ts DESC;


-- Prior firing/resolved state per (rule, agent). Restart-safe: this table is
-- the persisted alert state.
-- name: GetLatestEventPerTarget :many
SELECT DISTINCT ON (rule_id, agent_id) rule_id,
                                       agent_id,
                                       state
FROM alert_events
ORDER BY rule_id, agent_id, created_at DESC;


-- name: InsertAlertEvent :one
INSERT INTO alert_events (rule_id, agent_id, state, value, webhook_status)
VALUES (@rule_id, @agent_id, @state, @value, @webhook_status)
RETURNING *;


-- name: UpdateEventWebhookStatus :exec
UPDATE alert_events
SET webhook_status = @webhook_status,
    webhook_error  = @webhook_error
WHERE id = @id;


-- name: ListActiveAlerts :many
SELECT e.id,
       e.rule_id        AS "ruleId",
       r.name           AS "ruleName",
       r.metric         AS "metric",
       e.agent_id       AS "agentId",
       e.state          AS "state",
       e.value          AS "value",
       e.created_at     AS "createdAt",
       e.webhook_status AS "webhookStatus",
       e.webhook_error  AS "webhookError"
FROM (SELECT DISTINCT ON (rule_id, agent_id) *
      FROM alert_events
      ORDER BY rule_id, agent_id, created_at DESC) e
         JOIN alert_rules r ON r.id = e.rule_id
WHERE e.state = 'firing'
ORDER BY e.created_at DESC;


-- name: ListAlertEvents :many
SELECT e.id,
       e.rule_id        AS "ruleId",
       r.name           AS "ruleName",
       r.metric         AS "metric",
       e.agent_id       AS "agentId",
       e.state          AS "state",
       e.value          AS "value",
       e.created_at     AS "createdAt",
       e.webhook_status AS "webhookStatus",
       e.webhook_error  AS "webhookError"
FROM alert_events e
         JOIN alert_rules r ON r.id = e.rule_id
ORDER BY e.created_at DESC
LIMIT @lim;