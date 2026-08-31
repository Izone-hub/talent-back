-- +goose Up
-- +goose StatementBegin

CREATE TYPE application_status AS ENUM (
    'draft',
    'submitted',
    'quiz_started',
    'quiz_completed',
    'under_review',
    'shortlisted',
    'interviewed',
    'rejected',
    'accepted',
    'withdrawn'
);


-- job applications table
CREATE TABLE IF NOT EXISTS job_applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Foreign keys
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

        -- Snapshot of applicant data at time of application
    github_username VARCHAR(255) NOT NULL,
    github_id BIGINT NOT NULL,
    applicant_email VARCHAR(255),
    applicant_name VARCHAR(255),
    applicant_avatar_url TEXT,
    
    -- Application details
    cover_letter TEXT,
    proposed_salary INTEGER,           -- Expected salary
    proposed_salary_currency VARCHAR(3) DEFAULT 'ETB',
    availability_date TIMESTAMP,
    portfolio_url VARCHAR(500),         -- Personal website/portfolio
    linkedin_url VARCHAR(500),
    notes TEXT,                         -- Additional info from applicant
    
    -- Status tracking
    status application_status NOT NULL DEFAULT 'draft',
    submitted_at TIMESTAMP,              -- When they officially applied
    
    -- Review tracking (set by employer)
    reviewed_at TIMESTAMP,
    reviewed_by UUID REFERENCES users(id),
    employer_feedback TEXT,              -- Feedback from employer
    rejection_reason VARCHAR(255),       -- Why rejected (for analytics)
    
    -- Quiz tracking
    quiz_id UUID,                        -- Which quiz assigned
    quiz_score INTEGER,                  -- Score out of 100
    quiz_completed_at TIMESTAMP,
    quiz_passed BOOLEAN,                  -- Whether they passed
    
    -- AI summary access control
    can_view_ai_summary BOOLEAN DEFAULT false, -- Only employers can see AI summary
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT one_application_per_job UNIQUE(job_id, user_id),
    CONSTRAINT valid_submission CHECK (
        (status = 'draft' AND submitted_at IS NULL) OR
        (status != 'draft' AND submitted_at IS NOT NULL)
    )
);

-- Indexes for performance
CREATE INDEX idx_applications_user_id ON job_applications(user_id);
CREATE INDEX idx_applications_job_id ON job_applications(job_id);
CREATE INDEX idx_applications_status ON job_applications(status);
CREATE INDEX idx_applications_submitted_at ON job_applications(submitted_at);
CREATE INDEX idx_applications_quiz_id ON job_applications(quiz_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS job_applications;
DROP TYPE IF EXISTS application_status;
-- +goose StatementEnd