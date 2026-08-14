-- +goose Up
-- +goose StatementBegin
-- Make (user_id, version) unique so we can reference it
ALTER TABLE cv_versions ADD CONSTRAINT cv_versions_user_id_version_key UNIQUE (user_id, version);

-- Add cv_version and the composite foreign key
ALTER TABLE ai_summaries ADD COLUMN cv_version INT;
ALTER TABLE ai_summaries ADD CONSTRAINT fk_ai_summaries_cv
  FOREIGN KEY (user_id, cv_version) REFERENCES cv_versions(user_id, version) ON UPDATE CASCADE ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ai_summaries DROP CONSTRAINT IF EXISTS fk_ai_summaries_cv;
ALTER TABLE ai_summaries DROP COLUMN IF EXISTS cv_version;
ALTER TABLE cv_versions DROP CONSTRAINT IF EXISTS cv_versions_user_id_version_key;
-- +goose StatementEnd