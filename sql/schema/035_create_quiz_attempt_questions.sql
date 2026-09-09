-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS quiz_attempt_questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    quiz_attempt_id UUID NOT NULL
        REFERENCES quiz_attempts(id) ON DELETE CASCADE,

    question_id UUID NOT NULL
        REFERENCES questions(id) ON DELETE CASCADE,

    question_order INTEGER NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE (quiz_attempt_id, question_order),
    UNIQUE (quiz_attempt_id, question_id)
);

CREATE INDEX IF NOT EXISTS idx_quiz_attempt_questions_question_id ON quiz_attempt_questions(question_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS quiz_attempt_questions;
-- +goose StatementEnd
