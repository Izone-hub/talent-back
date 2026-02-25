-- +goose Up
-- +goose StatementBegin

-- Saved jobs for bookmarking (requires auth)
CREATE TABLE IF NOT EXISTS saved_jobs (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    saved_at TIMESTAMP NOT NULL DEFAULT NOW(),
    notes TEXT,                          -- Personal notes about the job
    
    PRIMARY KEY (user_id, job_id)
);

CREATE INDEX idx_saved_jobs_user_id ON saved_jobs(user_id);
CREATE INDEX idx_saved_jobs_saved_at ON saved_jobs(saved_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS saved_jobs;
-- +goose StatementEnd