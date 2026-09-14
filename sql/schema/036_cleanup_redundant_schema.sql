-- +goose Up
-- +goose StatementBegin

-- 1. Drop bloat and unused columns from the users table
ALTER TABLE users DROP COLUMN IF EXISTS twitter_username;
ALTER TABLE users DROP COLUMN IF EXISTS contribution_count;
ALTER TABLE users DROP COLUMN IF EXISTS hireable;
ALTER TABLE users DROP COLUMN IF EXISTS blog;
ALTER TABLE users DROP COLUMN IF EXISTS company;
ALTER TABLE users DROP COLUMN IF EXISTS public_gists;
ALTER TABLE users DROP COLUMN IF EXISTS location;

-- 2. Drop abandoned sandbox / code judge prototype tables (0 queries, 0 services)
DROP TABLE IF EXISTS submissions;
DROP TABLE IF EXISTS sandbox_templates;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Restore dropped columns on users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS twitter_username VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS contribution_count INTEGER DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS hireable BOOLEAN DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS blog VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS company VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS public_gists INTEGER DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS location VARCHAR(255);

-- +goose StatementEnd
