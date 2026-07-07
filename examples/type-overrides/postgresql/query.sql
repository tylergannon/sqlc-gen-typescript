-- name: GetUser :one
SELECT
  id,
  favorite_team_id,
  email,
  birthday,
  invited_at,
  login_count,
  quota_bytes,
  metadata
FROM users
WHERE id = $1;

-- name: ListUsersByFavoriteTeam :many
SELECT
  id,
  favorite_team_id,
  email,
  birthday,
  invited_at,
  login_count,
  quota_bytes,
  metadata
FROM users
WHERE favorite_team_id = $1
ORDER BY birthday;

-- name: CreateUser :one
INSERT INTO users (
  id,
  favorite_team_id,
  email,
  birthday,
  invited_at,
  login_count,
  quota_bytes,
  metadata
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING
  id,
  favorite_team_id,
  email,
  birthday,
  invited_at,
  login_count,
  quota_bytes,
  metadata;

-- name: UpdateUserBirthday :one
UPDATE users
SET birthday = $2
WHERE id = $1
RETURNING id, birthday, invited_at;
