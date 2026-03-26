-- name: DeleteVerifyTokenFromEmail :exec
DELETE FROM verify_tokens WHERE email = $1;

-- name: CreateVerifyToken :one
INSERT INTO verify_tokens( token, created_at, updated_at, email, expires_at, used)
values(
    $1,
    $2,
    $3,
    $4,
    $5,
    FALSE)
RETURNING *;

-- name: GetVerifyToken :one
SELECT * FROM verify_tokens WHERE token = $1 LIMIT 1;

-- name: GetVerifyTokenFromId :one
SELECT * FROM verify_tokens WHERE email = $1 LIMIT 1;

-- name: UseVerifyToken :exec
UPDATE verify_tokens 
SET used = 1, updated_at = $1 WHERE token = $1;

-- name: ResetVerifyTokens :exec
DELETE FROM verify_tokens;