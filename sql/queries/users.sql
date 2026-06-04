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

-- name: EditUserPwd :exec
UPDATE users 
SET updated_at=NOW(), hashed_password=$1
WHERE id = $2;

-- name: EditUserEmail :exec
UPDATE users 
SET updated_at=NOW(), email=$1
WHERE id = $2;
