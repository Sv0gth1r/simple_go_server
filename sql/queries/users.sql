-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
	gen_random_uuid(),
	NOW(),
	NOW(),
	$1,
	$2
)
RETURNING *;

-- name: DeleteAllUsers :exec
DELETE FROM users WHERE id IS NOT NULL;

-- name: GetUserPwdHash :one
SELECT hashed_password FROM users WHERE email = $1;

-- name: GetUserFromEmail :one
SELECT * FROM users WHERE email = $1;
