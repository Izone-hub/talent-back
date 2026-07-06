-- name: CreateRepositoryAnalysis :one
INSERT INTO repository_analysis (
    user_id,
    repo_name,
    language,
    score,
    has_readme,
    stars,
    forks,
    signals
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetRepositoryAnalysisByUser :many
SELECT * FROM repository_analysis
WHERE user_id = $1
ORDER BY analyzed_at DESC;

-- name: GetLatestRepositoryAnalysisByRepo :one
SELECT * FROM repository_analysis
WHERE user_id = $1 AND repo_name = $2
ORDER BY analyzed_at DESC
LIMIT 1;

-- name: DeleteRepositoryAnalysisByUser :exec
DELETE FROM repository_analysis
WHERE user_id = $1;
