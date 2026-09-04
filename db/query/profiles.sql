-- name: DeleteExpiredProfileCaptures :execrows
DELETE FROM profile_captures
WHERE expires_at <= NOW();

-- name: CreateProfileCapture :one
WITH expired AS (
    DELETE FROM profile_captures WHERE expires_at <= NOW()
)
INSERT INTO profile_captures (
    id, agent_id, target_name, profile_type, duration_seconds
) VALUES (
    @id, @agent_id, @target_name, @profile_type, @duration_seconds
)
RETURNING *;

-- name: GetProfileCapture :one
WITH expired AS (
    DELETE FROM profile_captures WHERE expires_at <= NOW()
)
SELECT *
FROM profile_captures
WHERE profile_captures.id = @id;

-- name: ListProfileCapturesByAgent :many
WITH expired AS (
    DELETE FROM profile_captures WHERE expires_at <= NOW()
)
SELECT *
FROM profile_captures
WHERE profile_captures.agent_id = @agent_id
ORDER BY created_at DESC
LIMIT @lim;

-- name: StartProfileCapture :one
WITH expired AS (
    DELETE FROM profile_captures WHERE expires_at <= NOW()
)
UPDATE profile_captures
SET state = 'capturing', updated_at = NOW()
WHERE profile_captures.id = @id
  AND profile_captures.agent_id = @agent_id
  AND state = 'queued'
  AND expires_at > NOW()
RETURNING *;

-- name: CompleteProfileCapture :one
WITH expired AS (
    DELETE FROM profile_captures WHERE expires_at <= NOW()
)
UPDATE profile_captures
SET state = 'completed',
    filename = @filename,
    content_type = @content_type,
    artifact = @artifact,
    error = NULL,
    updated_at = NOW(),
    expires_at = NOW() + INTERVAL '24 hours'
WHERE profile_captures.id = @id
  AND profile_captures.agent_id = @agent_id
  AND state IN ('queued', 'capturing')
  AND expires_at > NOW()
RETURNING *;

-- name: FailProfileCapture :one
WITH expired AS (
    DELETE FROM profile_captures WHERE expires_at <= NOW()
)
UPDATE profile_captures
SET state = 'failed',
    error = @error,
    updated_at = NOW(),
    expires_at = NOW() + INTERVAL '24 hours'
WHERE profile_captures.id = @id
  AND profile_captures.agent_id = @agent_id
  AND state IN ('queued', 'capturing')
  AND expires_at > NOW()
RETURNING *;
