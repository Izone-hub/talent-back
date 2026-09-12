-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS cv_analysis_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cv_version INTEGER NOT NULL DEFAULT 1,
    file_path TEXT NOT NULL,
    file_name TEXT NOT NULL,
    github_username TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'pending' 
        CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    last_error TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cv_analysis_jobs_queue 
    ON cv_analysis_jobs(status, created_at) 
    WHERE status IN ('pending', 'processing');

CREATE INDEX IF NOT EXISTS idx_cv_analysis_jobs_user 
    ON cv_analysis_jobs(user_id, cv_version);

-- Enforce at most one active (pending/processing) analysis job per (user_id, cv_version)
CREATE UNIQUE INDEX IF NOT EXISTS idx_cv_analysis_jobs_active_unique 
    ON cv_analysis_jobs(user_id, cv_version) 
    WHERE status IN ('pending', 'processing');

-- Idempotency constraint: ensure at most one ai_summary per (user_id, cv_version)
CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_summaries_user_cv_version 
    ON ai_summaries(user_id, cv_version) 
    WHERE cv_version IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_ai_summaries_user_cv_version;
DROP TABLE IF EXISTS cv_analysis_jobs;

-- +goose StatementEnd
