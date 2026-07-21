-- +goose Up
-- +goose StatementBegin
ALTER TABLE cv_versions ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT NOW();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE cv_versions DROP COLUMN IF EXISTS updated_at;
-- +goose StatementEnd
