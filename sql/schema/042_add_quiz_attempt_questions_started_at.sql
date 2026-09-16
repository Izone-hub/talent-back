-- +goose Up
-- +goose StatementBegin
ALTER TABLE quiz_attempt_questions ADD COLUMN IF NOT EXISTS started_at TIMESTAMP;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE quiz_attempt_questions DROP COLUMN IF EXISTS started_at;
-- +goose StatementEnd
