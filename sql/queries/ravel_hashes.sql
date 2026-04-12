-- name: ResetRavelHashes :exec
DELETE FROM ravel_hashes;

-- name: GetRavelHashes :many
SELECT image_path, ravel_post, full_hash, ravel_id, permalink, post_name
FROM ravel_hashes
WHERE hash_part_1 = $1 
   OR hash_part_2 = $2 
   OR hash_part_3 = $3 
   OR hash_part_4 = $4; 
-- name: CreateRavelHash :one
INSERT INTO ravel_hashes(image_path, ravel_post, full_hash,hash_part_1,hash_part_2, hash_part_3,hash_part_4, ravel_id, permalink, post_name)
values(
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10)
RETURNING *;