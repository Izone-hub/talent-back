-- +goose Up
-- +goose StatementBegin

-- Job category enum for role classification
CREATE TYPE job_category AS ENUM (
    'full_stack_developer',
    'web_developer',
    'frontend_developer',
    'backend_developer',
    'system_architect',
    'mobile_developer'
);

-- Add category column to jobs table with a sensible default
ALTER TABLE jobs ADD COLUMN category job_category NOT NULL DEFAULT 'full_stack_developer';

-- Index for filtering jobs by category
CREATE INDEX idx_jobs_category ON jobs(category);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_jobs_category;
ALTER TABLE jobs DROP COLUMN IF EXISTS category;
DROP TYPE IF EXISTS job_category;

-- +goose StatementEnd
