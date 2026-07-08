-- name: CreateUserSkillProfile :one
INSERT INTO user_skill_profile (
    user_id,
    backend_score,
    frontend_score,
    devops_score,
    database_score,
    backend_level,
    frontend_level,
    devops_level,
    overall_score
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetLatestUserSkillProfile :one
SELECT * FROM user_skill_profile
WHERE user_id = $1
ORDER BY generated_at DESC
LIMIT 1;
