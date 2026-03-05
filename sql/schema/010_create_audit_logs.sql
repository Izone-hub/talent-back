-- +goose Up
-- +goose StatementBegin

-- Audit action types
CREATE TYPE audit_action AS ENUM (
    -- User actions
    'user_login',
    'user_logout',
    'user_registered',
    'user_updated',
    'user_deleted',
    'role_changed',
    
    -- Job actions
    'job_created',
    'job_updated',
    'job_published',
    'job_closed',
    'job_archived',
    'job_deleted',
    
    -- Application actions
    'application_submitted',
    'application_updated',
    'application_status_changed',
    'application_withdrawn',
    
    -- Quiz actions
    'quiz_started',
    'quiz_completed',
    'quiz_auto_saved',
    'quiz_timed_out',
    
    -- CV actions
    'cv_uploaded',
    'cv_deleted',
    'cv_updated',
    
    -- Admin actions
    'admin_login',
    'admin_action',
    'settings_changed',
    'report_generated'
);

-- Audit logs table (index on performed_at for time-range queries)
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Who performed the action
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    user_email VARCHAR(255),
    user_role VARCHAR(50),
    
    -- What action was performed
    action audit_action NOT NULL,
    entity_type VARCHAR(50), -- 'user', 'job', 'application', 'quiz', 'cv'
    entity_id UUID, -- ID of the affected record
    
    -- When and where
    performed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    
    -- Details (JSON for flexibility)
    details JSONB,
    
    -- Before/after state for updates
    old_values JSONB,
    new_values JSONB,
    
    -- Status
    status VARCHAR(20) DEFAULT 'success', -- 'success', 'failure', 'pending'
    error_message TEXT,
    
    -- For compliance (GDPR, etc.)
    data_retention_days INTEGER DEFAULT 365, -- How long to keep
    marked_for_deletion BOOLEAN DEFAULT false
);

-- Indexes
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_performed_at ON audit_logs(performed_at);
CREATE INDEX idx_audit_logs_ip ON audit_logs(ip_address);

-- Admin audit trail (for sensitive actions)
CREATE TABLE IF NOT EXISTS admin_audit_trail (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    audit_log_id UUID NOT NULL REFERENCES audit_logs(id) ON DELETE CASCADE,
    
    -- Additional admin-specific info
    admin_id UUID NOT NULL REFERENCES users(id),
    admin_notes TEXT,
    approval_required BOOLEAN DEFAULT false,
    approved_by UUID REFERENCES users(id),
    approved_at TIMESTAMP,
    
    -- Sensitivity level
    sensitivity_level VARCHAR(20) DEFAULT 'medium', -- 'low', 'medium', 'high', 'critical'
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Compliance exports tracking
CREATE TABLE IF NOT EXISTS audit_exports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exported_by UUID REFERENCES users(id),
    export_type VARCHAR(50), -- 'gdpr', 'investigation', 'report'
    date_range_start TIMESTAMP,
    date_range_end TIMESTAMP,
    filter_criteria JSONB,
    file_path TEXT,
    exported_at TIMESTAMP NOT NULL DEFAULT NOW(),
    downloaded_count INTEGER DEFAULT 0
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit_exports;
DROP TABLE IF EXISTS admin_audit_trail;
DROP TABLE IF EXISTS audit_logs;
DROP TYPE IF EXISTS audit_action;
-- +goose StatementEnd