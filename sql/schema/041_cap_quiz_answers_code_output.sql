-- +goose Up
-- +goose StatementBegin
-- Enforce a hard 32 KB (32768 characters) ceiling on code_output to prevent database text bloat
ALTER TABLE quiz_answers ADD CONSTRAINT check_quiz_answers_code_output_length 
    CHECK (code_output IS NULL OR length(code_output) <= 32768);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE quiz_answers DROP CONSTRAINT IF EXISTS check_quiz_answers_code_output_length;
-- +goose StatementEnd
