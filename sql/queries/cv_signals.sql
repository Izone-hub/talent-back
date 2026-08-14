-- name: UpsertCVSignals :one
INSERT INTO cv_signals (
    user_id,
    claimed_skills,
    experience_level,
    projects_listed,
    credibility,
    alignment_with_github,
    raw_summary
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id)
DO UPDATE SET
    claimed_skills = EXCLUDED.claimed_skills,
    experience_level = EXCLUDED.experience_level,
    projects_listed = EXCLUDED.projects_listed,
    credibility = EXCLUDED.credibility,
    alignment_with_github = EXCLUDED.alignment_with_github,
    raw_summary = EXCLUDED.raw_summary,
    updated_at = NOW()
RETURNING *;

-- name: GetCVSignalsByUser :one
SELECT * FROM cv_signals
WHERE user_id = $1
LIMIT 1;
