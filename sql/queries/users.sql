-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
    VALUES (gen_random_uuid (), Now(), Now(), $1, $2)
RETURNING
    id, created_at, updated_at, email;

-- name: RemoveUsers :exec
DELETE FROM users;

-- name: GetUserByEmail :one
SELECT
    *
FROM
    users
WHERE
    users.email = $1;

-- name: UpdateUser :one
UPDATE
    users
SET
    email = $2,
    hashed_password = $3,
    updated_at = Now()
WHERE
    id = $1
RETURNING
    *;

