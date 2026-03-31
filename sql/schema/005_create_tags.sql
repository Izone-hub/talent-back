-- +goose Up
-- +goose StatementBegin

CREATE TYPE tag_category AS ENUM ('skill');

-- Tags table - reusable tags for jobs, questions, etc.
CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    category tag_category,
    description TEXT,
    color VARCHAR(7), -- Hex color code for UI
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Junction table for jobs and tags (many-to-many)
CREATE TABLE IF NOT EXISTS job_tags (
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (job_id, tag_id)
);

-- Indexes
CREATE INDEX idx_tags_category ON tags(category);
CREATE INDEX idx_tags_name ON tags(name);
CREATE INDEX idx_job_tags_tag_id ON job_tags(tag_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS job_tags;
DROP TABLE IF EXISTS tags;
-- +goose StatementEnd