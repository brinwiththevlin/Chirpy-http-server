-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email)
    VALUES (gen_random_uuid (), Now(), Now(), $1)
RETURNING
    *;

-- name: RemoveUsers :exec
DELETE FROM usres;

