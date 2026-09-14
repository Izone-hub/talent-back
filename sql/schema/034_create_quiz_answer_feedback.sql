-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS quiz_answer_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    quiz_attempt_id UUID NOT NULL REFERENCES quiz_attempts(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    application_id UUID NOT NULL REFERENCES job_applications(id) ON DELETE CASCADE,
    feedback VARCHAR(30) NOT NULL CHECK (feedback IN ('not_related', 'not_enough_time', 'too_difficult', 'unclear_question')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, quiz_attempt_id, question_id)
);

CREATE INDEX IF NOT EXISTS idx_quiz_answer_feedback_attempt_id ON quiz_answer_feedback(quiz_attempt_id);
CREATE INDEX IF NOT EXISTS idx_quiz_answer_feedback_application_id ON quiz_answer_feedback(application_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS quiz_answer_feedback;
-- +goose StatementEnd
