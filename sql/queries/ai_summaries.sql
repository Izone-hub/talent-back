-- name: CreateAISummary :one
INSERT INTO ai_summaries (user_id, summary, strengths, weaknesses, model, cv_version)
VALUES ($1, $2, $3, $4, $5, $6)

RETURNING *;

-- name: GetLatestAISummary :one
SELECT * FROM ai_summaries
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: GetAISummaryByCVVersion :one
SELECT * FROM ai_summaries
WHERE user_id = $1 AND cv_version = $2;
