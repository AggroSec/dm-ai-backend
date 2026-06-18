-- name: GetUserByUsername :one

select * from users where username = $1;

-- name: CreateUser :one

INSERT INTO users (username, hashed_password)
values ($1, $2)
RETURNING *;

-- name: GetUserByID :one

select * from users WHERE id = $1;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1;

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens
WHERE token = $1;

-- name: DeleteUserRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE user_id = $1;