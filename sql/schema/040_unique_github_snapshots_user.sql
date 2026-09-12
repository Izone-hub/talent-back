-- +goose Up
-- +goose StatementBegin
ALTER TABLE github_snapshots ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT NOW();

-- Keep only the latest snapshot per user if duplicates exist
DELETE FROM github_snapshots
WHERE id NOT IN (
    SELECT DISTINCT ON (user_id) id
    FROM github_snapshots
    WHERE user_id IS NOT NULL
    ORDER BY user_id, fetched_at DESC NULLS LAST
) AND user_id IS NOT NULL;

-- Add unique constraint to guarantee at most one snapshot per user
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'unique_github_snapshots_user_id'
    ) THEN
        ALTER TABLE github_snapshots ADD CONSTRAINT unique_github_snapshots_user_id UNIQUE (user_id);
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE github_snapshots DROP CONSTRAINT IF EXISTS unique_github_snapshots_user_id;
ALTER TABLE github_snapshots DROP COLUMN IF EXISTS updated_at;
-- +goose StatementEnd
