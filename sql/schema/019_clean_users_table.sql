-- +goose Up
-- +goose StatementBegin
ALTER TABLE users 
  DROP COLUMN IF EXISTS public_repos,
  DROP COLUMN IF EXISTS public_gists,
  DROP COLUMN IF EXISTS followers,
  DROP COLUMN IF EXISTS following,
  DROP COLUMN IF EXISTS top_languages,
  DROP COLUMN IF EXISTS contribution_count;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users 
  ADD COLUMN IF NOT EXISTS public_repos INTEGER DEFAULT 0,
  ADD COLUMN IF NOT EXISTS public_gists INTEGER DEFAULT 0,
  ADD COLUMN IF NOT EXISTS followers INTEGER DEFAULT 0,
  ADD COLUMN IF NOT EXISTS following INTEGER DEFAULT 0,
  ADD COLUMN IF NOT EXISTS top_languages TEXT[] DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS contribution_count INTEGER DEFAULT 0;
-- +goose StatementEnd
