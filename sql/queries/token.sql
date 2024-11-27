-- name: CreateRefreshToken :one
INSERT INTO refresh_token (token, expires_at, user_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshTokenByToken :one
SELECT * FROM refresh_token
WHERE token = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_token
SET revoked_at = NOW()
WHERE token = $1;
