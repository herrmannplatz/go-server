-- name: CreateChirp :one
INSERT INTO chirps (body, user_id)
VALUES ($1, $2)
RETURNING *;

-- name: GetChirps :many
SELECT * FROM chirps ORDER BY created_at ASC;

-- name: GetChirp :one
SELECT * FROM chirps WHERE id = $1;

-- name: DeleteChirp :exec
DELETE FROM chirps 
WHERE id = $1 AND user_id = $2;

-- name: GetChirpsWithOptions :many
SELECT * FROM chirps
WHERE user_id = COALESCE($1, user_id)
ORDER BY 
  CASE WHEN $2 = 'desc' THEN created_at END DESC,
  CASE WHEN $2 = 'asc' THEN created_at END ASC;
