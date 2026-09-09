-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS quiz_result_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    quiz_attempt_id UUID NOT NULL REFERENCES quiz_attempts(id) ON DELETE CASCADE,
    rating VARCHAR(10) NOT NULL CHECK (rating IN ('positive', 'negative')),
    comment TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, quiz_attempt_id)
);

CREATE INDEX IF NOT EXISTS idx_quiz_result_feedback_attempt_id ON quiz_result_feedback(quiz_attempt_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS quiz_result_feedback;
-- +goose StatementEnd
