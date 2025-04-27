-- name: GetRefreshToken :one
SELECT
    *
FROM
    refresh_tokens
WHERE
    token = $1;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, created_at, updated_at, expires_at, token)
    VALUES ($1, Now(), Now(), $2, $3)
RETURNING
    *;

-- name: RevokeRefreshToken :exec
UPDATE
    refresh_tokens
SET
    revoked_at = Now(),
    updated_at = Now()
WHERE
    token = $1;

