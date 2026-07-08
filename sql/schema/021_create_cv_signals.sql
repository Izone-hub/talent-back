-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS cv_signals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  claimed_skills JSONB DEFAULT '[]'::jsonb,
  experience_level VARCHAR(50),
  projects_listed INTEGER DEFAULT 0,
  credibility VARCHAR(50),
  alignment_with_github VARCHAR(50),
  raw_summary TEXT,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  UNIQUE (user_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS cv_signals;
-- +goose StatementEnd
