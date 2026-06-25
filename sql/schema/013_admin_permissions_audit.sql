-- +goose Up
-- +goose StatementBegin

-- Add audit_action values for question-level changes (safe-add)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e
        JOIN pg_type t ON e.enumtypid = t.oid
        WHERE t.typname = 'audit_action' AND e.enumlabel = 'question_created'
    ) THEN
        ALTER TYPE audit_action ADD VALUE 'question_created';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e
        JOIN pg_type t ON e.enumtypid = t.oid
        WHERE t.typname = 'audit_action' AND e.enumlabel = 'question_updated'
    ) THEN
        ALTER TYPE audit_action ADD VALUE 'question_updated';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e
        JOIN pg_type t ON e.enumtypid = t.oid
        WHERE t.typname = 'audit_action' AND e.enumlabel = 'question_deleted'
    ) THEN
        ALTER TYPE audit_action ADD VALUE 'question_deleted';
    END IF;
END$$;

-- Extend admins.permissions default to include read-only applicant summaries access
ALTER TABLE admins
    ALTER COLUMN permissions SET DEFAULT (
        '{
            "manage_jobs": true,
            "manage_questions": true,
            "manage_tags": true,
            "manage_weights": true,
            "manage_difficulties": true,
            "view_applicants": true,
            "view_cvs": true,
            "view_github_metadata": true,
            "view_ai_summaries": true,
            "view_applicant_summaries": true,
            "view_audit_logs": true
        }'::jsonb
    );

-- Ensure existing admin rows have the new key set to true if unset
UPDATE admins
SET permissions = permissions || '{"view_applicant_summaries": true}'::jsonb
WHERE NOT (permissions ? 'view_applicant_summaries');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Revert default to previous shape (leave enum values in place — enums cannot be easily removed)
ALTER TABLE admins
    ALTER COLUMN permissions SET DEFAULT (
        '{
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
        }'::jsonb
    );

-- Optionally remove the value from rows
UPDATE admins
SET permissions = permissions - 'view_applicant_summaries'
WHERE (permissions ? 'view_applicant_summaries');

-- +goose StatementEnd
