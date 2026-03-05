-- +goose Up
-- +goose StatementBegin

-- CV versions tracking (PDF only as required)
CREATE TABLE IF NOT EXISTS cv_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Ownership
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- File info (PDF only)
    file_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,      -- Storage path
    file_size INTEGER NOT NULL,            -- In bytes
    file_hash VARCHAR(64) NOT NULL,        -- SHA-256 for integrity check
    
    -- Version tracking
    version INTEGER NOT NULL DEFAULT 1,
    is_current BOOLEAN DEFAULT true,
    
    -- Metadata
    uploaded_at TIMESTAMP NOT NULL DEFAULT NOW(),
    uploaded_from_ip INET,                  -- For audit
    
    -- Application reference (which application used this CV)
    application_id UUID REFERENCES job_applications(id) ON DELETE SET NULL,
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Ensure PDF only
    CONSTRAINT valid_cv_format CHECK (file_name ~* '.*\.pdf$')
);

-- Track CV usage across applications
CREATE TABLE IF NOT EXISTS cv_application_usage (
    cv_id UUID NOT NULL REFERENCES cv_versions(id) ON DELETE CASCADE,
    application_id UUID NOT NULL REFERENCES job_applications(id) ON DELETE CASCADE,
    used_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (cv_id, application_id)
);

-- Indexes
CREATE INDEX idx_cv_versions_user_id ON cv_versions(user_id);
CREATE INDEX idx_cv_versions_is_current ON cv_versions(is_current) WHERE is_current = true;
CREATE INDEX idx_cv_versions_application_id ON cv_versions(application_id);
CREATE INDEX idx_cv_versions_file_hash ON cv_versions(file_hash); -- For detecting duplicates

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS cv_application_usage;
DROP TABLE IF EXISTS cv_versions;
-- +goose StatementEnd