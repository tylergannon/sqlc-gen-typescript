-- name: GetAuthor :one
SELECT * FROM authors
WHERE id = $1 LIMIT 1;

-- name: ListAuthors :many
SELECT * FROM authors
ORDER BY name;

-- name: CreateAuthor :one
INSERT INTO authors (
  name, bio, status, profile, notes
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: ListAuthorsByStatus :many
SELECT * FROM authors
WHERE status = $1
ORDER BY name;

-- name: DeleteAuthor :exec
DELETE FROM authors
WHERE id = $1;
