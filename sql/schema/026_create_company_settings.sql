-- +goose Up
-- +goose StatementBegin

CREATE TABLE company_settings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_name    TEXT NOT NULL DEFAULT '',
    company_logo    TEXT NOT NULL DEFAULT '',
    company_website TEXT NOT NULL DEFAULT '',
    company_location TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Insert default row so the table is never empty
INSERT INTO company_settings (company_name, company_location)
VALUES ('iZone Hub', 'Addis Ababa, Ethiopia');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS company_settings;

-- +goose StatementEnd
