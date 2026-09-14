-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_quiz_attempts_job_id ON quiz_attempts(job_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_quiz_attempts_job_id;
-- +goose StatementEnd
