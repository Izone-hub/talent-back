-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    github_id BIGINT UNIQUE NOT NULL,
    github_username VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    avatar_url TEXT,
    name VARCHAR(255),
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    github_access_token TEXT,
    github_token_expires_at TIMESTAMP,
    last_login_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd