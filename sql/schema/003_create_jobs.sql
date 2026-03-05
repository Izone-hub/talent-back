-- +goose Up
-- +goose StatementBegin

-- Job status enum (only what's directly related to jobs)
CREATE TYPE job_status AS ENUM ('draft', 'published', 'closed', 'archived');
CREATE TYPE job_type AS ENUM ('full-time', 'part-time', 'contract', 'freelance', 'internship');
CREATE TYPE job_experience_level AS ENUM ('entry', 'junior', 'mid', 'senior', 'lead');

-- Jobs table - ONLY job-specific fields
CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Core job information
    title VARCHAR(255) NOT NULL,
    company VARCHAR(255) NOT NULL,
    company_logo TEXT,
    company_website VARCHAR(255),
    company_location VARCHAR(255),
    description TEXT NOT NULL,
    requirements TEXT NOT NULL,
    responsibilities TEXT,
    benefits TEXT,
    
    -- Job specifications
    job_type job_type NOT NULL DEFAULT 'full-time',
    experience_level job_experience_level NOT NULL DEFAULT 'mid',
    location VARCHAR(255),
    remote_possible BOOLEAN DEFAULT false,
    salary_min INTEGER,
    salary_max INTEGER,
    salary_currency VARCHAR(3) DEFAULT 'USD',
    
    -- Status and lifecycle
    status job_status NOT NULL DEFAULT 'draft',
    published_at TIMESTAMP,
    closed_at TIMESTAMP,
    archived_at TIMESTAMP,
    expires_at TIMESTAMP,
    
    -- Foreign keys (only direct relationships)
    posted_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Tracking (counts that are updated by triggers/application)
    views_count INTEGER DEFAULT 0,
    applications_count INTEGER DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for jobs table
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_posted_by ON jobs(posted_by);
CREATE INDEX idx_jobs_published_at ON jobs(published_at) WHERE status = 'published';
CREATE INDEX idx_jobs_company ON jobs(company);
CREATE INDEX idx_jobs_location ON jobs(location);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS jobs;
DROP TYPE IF EXISTS job_experience_level;
DROP TYPE IF EXISTS job_type;
DROP TYPE IF EXISTS job_status;
-- +goose StatementEnd