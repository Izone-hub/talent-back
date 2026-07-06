-- +goose Up
-- Sandbox execution tables for code judge system

CREATE TABLE IF NOT EXISTS sandbox_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    language TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    template_path TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    question_id UUID REFERENCES questions(id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    code TEXT NOT NULL,
    execution_type TEXT NOT NULL DEFAULT 'standard',
    passed BOOLEAN DEFAULT false,
    stdout TEXT,
    stderr TEXT,
    exit_code INTEGER,
    time_ms INTEGER,
    error_message TEXT,
    submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_submissions_user_id ON submissions(user_id);
CREATE INDEX IF NOT EXISTS idx_submissions_question_id ON submissions(question_id);
CREATE INDEX IF NOT EXISTS idx_submissions_submitted_at ON submissions(submitted_at DESC);

-- +goose Down
DROP TABLE IF EXISTS submissions;
DROP TABLE IF EXISTS sandbox_templates;
