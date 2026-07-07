CREATE TABLE teams (
  id   uuid PRIMARY KEY,
  name text NOT NULL
);

CREATE TABLE users (
  id               uuid PRIMARY KEY,
  favorite_team_id uuid REFERENCES teams(id),
  email            text NOT NULL,
  birthday         text NOT NULL,
  invited_at       text,
  login_count      BIGINT NOT NULL DEFAULT 0,
  quota_bytes      BIGINT,
  metadata         jsonb
);
