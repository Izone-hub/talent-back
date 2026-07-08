-- name: CreateGitHubSnapshot :one
INSERT INTO github_snapshots (
    user_id,
    public_repos,
    followers,
    following,
    raw_data
)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetLatestGitHubSnapshot :one
SELECT * FROM github_snapshots
WHERE user_id = $1
ORDER BY fetched_at DESC
LIMIT 1;

-- name: ListGitHubSnapshots :many
SELECT * FROM github_snapshots
WHERE user_id = $1
ORDER BY fetched_at DESC;
