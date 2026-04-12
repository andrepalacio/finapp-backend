-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, name, created_at, updated_at)
VALUES (uuid_generate_v4(), $1, $2, $3, NOW(), NOW())
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;
