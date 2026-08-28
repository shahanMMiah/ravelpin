-- name: ResetUserSearch :exec
DELETE FROM user_search;

-- name: CreateUserSearch :one
INSERT INTO user_search(id, created_at, updated_at, user_id, search_img)
values(
$1,
$2,
$3,
$4,
$5)
RETURNING *;

-- name: GetUserSearches :many
SELECT * FROM user_search WHERE user_id = $1 ORDER BY created_at ASC;

-- name: ResetSearchResult :exec
DELETE FROM search_result;

-- name: CreateSearchResult :one
INSERT INTO search_result(id, name, permalink, image_path, search_id)
values(
$1,
$2,
$3,
$4,
$5)
RETURNING *;

-- name: GetUserSearchResults :many
SELECT * FROM search_result WHERE search_id = $1;
