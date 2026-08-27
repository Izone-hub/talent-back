-- name: CreateJob :one
INSERT INTO jobs (
    title, company, company_logo, company_website, company_location,
    description, requirements, responsibilities, benefits,
    job_type, category, experience_level, location, remote_possible,
    salary_min, salary_max, salary_currency,
    status, posted_by, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
    $15, $16, $17, $18, $19, $20
) RETURNING *;

-- name: GetJobByID :one
SELECT * FROM jobs WHERE id = $1;

-- name: GetPublishedJobByID :one
SELECT * FROM jobs 
WHERE id = $1 AND status = 'published';

-- name: ListJobsByPoster :many
SELECT * FROM jobs
WHERE posted_by = $1
  AND ($4::text = '' OR category = $4::job_category)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListPublishedJobs :many
SELECT j.*,
       jsonb_build_object(
           'applied', ja.id IS NOT NULL,
           'application_id', ja.id,
           'status', ja.status,
           'submitted_at', ja.submitted_at
       ) AS user_application
FROM jobs j
LEFT JOIN job_applications ja 
    ON ja.job_id = j.id AND ja.user_id = $4
WHERE j.status = 'published'
  AND ($1::text = '' OR j.category = $1::job_category)
ORDER BY j.published_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateJob :one
UPDATE jobs
SET 
    title = COALESCE($2, title),
    company = COALESCE($3, company),
    description = COALESCE($4, description),
    requirements = COALESCE($5, requirements),
    job_type = COALESCE($6, job_type),
    category = COALESCE($7, category),
    experience_level = COALESCE($8, experience_level),
    location = COALESCE($9, location),
    remote_possible = COALESCE($10, remote_possible),
    salary_min = COALESCE($11, salary_min),
    salary_max = COALESCE($12, salary_max),
    updated_at = NOW()
WHERE id = $1 AND posted_by = $13
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

-- name: ListAllPublishedJobs :many
SELECT 
    j.*,
    CASE 
        WHEN ja.id IS NOT NULL THEN 'applied'
        ELSE 'not_applied'
    END AS application_status
FROM jobs j 
LEFT JOIN job_applications ja 
    ON ja.job_id = j.id 
    AND ja.user_id = $4 
WHERE j.status = 'published'
  AND ($1::text = '' OR j.category = $1::job_category)
ORDER BY j.published_at DESC 
LIMIT $2 OFFSET $3;
