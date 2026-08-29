-- name: GetCompanySettings :one
SELECT * FROM company_settings LIMIT 1;

-- name: UpdateCompanySettings :one
UPDATE company_settings
SET
    company_name     = $1,
    company_logo     = $2,
    company_website  = $3,
    company_location = $4,
    updated_at       = NOW()
WHERE id = (SELECT id FROM company_settings LIMIT 1)
RETURNING *;

-- name: GetJobCountsByCategory :many
SELECT category, COUNT(*)::int AS job_count
FROM jobs
GROUP BY category
ORDER BY job_count DESC;
