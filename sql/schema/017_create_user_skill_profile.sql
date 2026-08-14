-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_skill_profile (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  backend_score INT DEFAULT 0,
  frontend_score INT DEFAULT 0,
  devops_score INT DEFAULT 0,
  database_score INT DEFAULT 0,
  backend_level TEXT,
  frontend_level TEXT,
  devops_level TEXT,
  overall_score INT DEFAULT 0,
  generated_at TIMESTAMP DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_skill_profile;
-- +goose StatementEnd
