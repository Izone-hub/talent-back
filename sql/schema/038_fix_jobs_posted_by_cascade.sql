-- +goose Up
-- +goose StatementBegin
ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_posted_by_fkey;
ALTER TABLE jobs ADD CONSTRAINT jobs_posted_by_fkey 
    FOREIGN KEY (posted_by) REFERENCES users(id) ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_posted_by_fkey;
ALTER TABLE jobs ADD CONSTRAINT jobs_posted_by_fkey 
    FOREIGN KEY (posted_by) REFERENCES users(id) ON DELETE CASCADE;
-- +goose StatementEnd
