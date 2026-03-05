-- name: CreateJob :one
INSERT INTO jobs (
    title, company, company_logo, company_website, company_location,
    description, requirements, responsibilities, benefits,
    job_type, experience_level, location, remote_possible,
    salary_min, salary_max, salary_currency,
    status, posted_by, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
    $14, $15, $16, $17, $18, $19
) RETURNING *;

-- name: GetJobByID :one
SELECT * FROM jobs WHERE id = $1;

-- name: GetPublishedJobByID :one
SELECT * FROM jobs 
WHERE id = $1 AND status = 'published';

-- name: ListJobsByPoster :many
SELECT * FROM jobs
WHERE posted_by = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListPublishedJobs :many
SELECT * FROM jobs
WHERE status = 'published'
ORDER BY published_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateJob :one
UPDATE jobs
SET 
    title = COALESCE($2, title),
    company = COALESCE($3, company),
    description = COALESCE($4, description),
    requirements = COALESCE($5, requirements),
    job_type = COALESCE($6, job_type),
    experience_level = COALESCE($7, experience_level),
    location = COALESCE($8, location),
    remote_possible = COALESCE($9, remote_possible),
    salary_min = COALESCE($10, salary_min),
    salary_max = COALESCE($11, salary_max),
    updated_at = NOW()
WHERE id = $1 AND posted_by = $12
RETURNING *;

-- Job lifecycle management
-- name: PublishJob :one
UPDATE jobs
SET status = 'published',
    published_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND posted_by = $2 AND status = 'draft'
RETURNING *;

-- name: CloseJob :one
UPDATE jobs
SET status = 'closed',
    closed_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND posted_by = $2 AND status = 'published'
RETURNING *;

-- name: ArchiveJob :one
UPDATE jobs
SET status = 'archived',
    archived_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND posted_by = $2 AND status IN ('published', 'closed')
RETURNING *;

-- name: IncrementJobViews :exec
UPDATE jobs SET views_count = views_count + 1 WHERE id = $1;