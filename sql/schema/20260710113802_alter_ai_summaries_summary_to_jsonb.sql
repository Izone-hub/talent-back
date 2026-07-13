-- +goose Up
ALTER TABLE ai_summaries ALTER COLUMN summary TYPE JSONB USING summary::jsonb;

-- +goose Down
ALTER TABLE ai_summaries ALTER COLUMN summary TYPE JSON USING summary::json;