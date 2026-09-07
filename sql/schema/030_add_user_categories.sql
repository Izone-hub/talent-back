-- +goose Up
-- +goose StatementBegin

-- Job categories a user belongs to (auto-derived from GitHub/AI analysis, e.g.
-- backend_developer, frontend_developer, full_stack_developer). A user can be
-- in several categories; only accepted users (acceptance_job_id IS NOT NULL)
-- are shown in the admin category cards.
ALTER TABLE users ADD COLUMN IF NOT EXISTS categories TEXT[] NOT NULL DEFAULT '{}';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE users DROP COLUMN IF EXISTS categories;

-- +goose StatementEnd