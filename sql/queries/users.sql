-- name: GetUserByUsername :one

select * from users where username = $1;

-- name: CreateUser :one

INSERT INTO users (username, hashed_password)
values ($1, $2)
RETURNING *;

-- name: GetUserByID :one

select * from users WHERE id = $1;