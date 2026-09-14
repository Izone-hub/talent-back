-- name: CreateGitHubSnapshot :one
INSERT INTO github_snapshots (
    user_id,
    public_repos,
    followers,
    following,
    raw_data,
    fetched_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
ON CONFLICT (user_id) DO UPDATE SET
    public_repos = EXCLUDED.public_repos,
    followers = EXCLUDED.followers,
    following = EXCLUDED.following,
    raw_data = EXCLUDED.raw_data,
    fetched_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetLatestGitHubSnapshot :one
SELECT * FROM github_snapshots
WHERE user_id = $1
LIMIT 1;

-- name: ListGitHubSnapshots :many
SELECT * FROM github_snapshots
WHERE user_id = $1
LIMIT 1;
