CREATE TYPE author_status AS ENUM ('active', 'inactive', 'pending');

CREATE TABLE authors (
  id     BIGSERIAL PRIMARY KEY,
  name   text          NOT NULL,
  bio    text,
  status author_status NOT NULL DEFAULT 'active',
  profile jsonb        NOT NULL DEFAULT '{}'::jsonb,
  notes   jsonb
);
