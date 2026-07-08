-- name: CreateAISummary :one
INSERT INTO ai_summaries (
    user_id,
    summary,
    strengths,
    weaknesses,
    model
)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetLatestAISummary :one
SELECT * FROM ai_summaries
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 1;
