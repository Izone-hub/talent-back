-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS repository_analysis (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  repo_name TEXT NOT NULL,
  language TEXT,
  score INT DEFAULT 0,
  has_readme BOOLEAN DEFAULT false,
  stars INT DEFAULT 0,
  forks INT DEFAULT 0,
  signals JSONB,
  analyzed_at TIMESTAMP DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS repository_analysis;
-- +goose StatementEnd
