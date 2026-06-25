-- +goose Up
-- +goose StatementBegin

-- Admins table: links (optionally) to `users` and stores granular permissions as JSONB.
CREATE TABLE IF NOT EXISTS admins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    github_id BIGINT UNIQUE,
    github_login VARCHAR(255),
    email VARCHAR(255),
    role VARCHAR(50) NOT NULL DEFAULT 'admin',
    permissions JSONB NOT NULL DEFAULT '{
        "manage_jobs": true,
        "manage_questions": true,
        "manage_tags": true,
        "manage_weights": true,
        "manage_difficulties": true,
        "view_applicants": true,
        "view_cvs": true,
        "view_github_metadata": true,
        "view_ai_summaries": true,
        "view_audit_logs": true
    }'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Allowlist of GitHub identities eligible to become admins (checked at OAuth login)
CREATE TABLE IF NOT EXISTS admin_allowlist (
    github_id BIGINT PRIMARY KEY,
    github_login VARCHAR(255),
    note TEXT,
    added_by UUID REFERENCES users(id) ON DELETE SET NULL,
    added_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admins_github_id ON admins(github_id);
CREATE INDEX IF NOT EXISTS idx_admin_allowlist_github_login ON admin_allowlist(github_login);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS admin_allowlist;
DROP TABLE IF EXISTS admins;
-- +goose StatementEnd
