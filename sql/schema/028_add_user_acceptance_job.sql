-- +goose Up
-- +goose StatementBegin

-- Tracks the single job a user has been accepted for.
-- NULL means the user has NOT accepted a job yet.
-- A non-NULL value means the user has been accepted for that specific job.
ALTER TABLE users ADD COLUMN IF NOT EXISTS acceptance_job_id UUID REFERENCES jobs(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_users_acceptance_job_id ON users(acceptance_job_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_users_acceptance_job_id;
ALTER TABLE users DROP COLUMN IF EXISTS acceptance_job_id;

-- +goose StatementEnd
