-- +goose Up
-- +goose StatementBegin

-- Screening survey questions attached to a job posting.
-- Admin creates yes/no questions; candidates must answer Yes to proceed.
CREATE TABLE job_survey_questions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    question_text   TEXT NOT NULL,
    expected_answer BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order      INT  NOT NULL DEFAULT 0,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_survey_questions_job_id ON job_survey_questions(job_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_survey_questions_job_id;
DROP TABLE IF EXISTS job_survey_questions;

-- +goose StatementEnd
