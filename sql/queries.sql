-- name: CreateUser :one
INSERT INTO users (id, first_name, last_name, email, password)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, email;

-- name: GetUserById :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByGithubID :one
SELECT * FROM users WHERE github_id = $1 LIMIT 1;

-- name: GetUserByGithubUsername :one
SELECT * FROM users WHERE github_username = $1 LIMIT 1;

-- name: CreateUserWithGithub :one
INSERT INTO users (
    id, first_name, last_name, email, github_username, github_id,
    avatar_url, auth_provider, github_access_token, bio, location,
    company, blog, role, talent_status, availability_status
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING *;

-- name: UpdateUserGithubData :one
UPDATE users
SET 
    github_username = COALESCE($2, github_username),
    avatar_url = COALESCE($3, avatar_url),
    github_access_token = COALESCE($4, github_access_token),
    bio = COALESCE($5, bio),
    location = COALESCE($6, location),
    company = COALESCE($7, company),
    blog = COALESCE($8, blog),
    tech_stack = COALESCE($9, tech_stack),
    last_active_at = COALESCE($10, last_active_at),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ListUsers :many
SELECT id, first_name, last_name, email, created_at, updated_at FROM users
ORDER BY id
LIMIT $1
OFFSET $2;

-- Job Categories Queries
-- name: CreateJobCategory :one
INSERT INTO job_categories (id, name, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetJobCategoryById :one
SELECT * FROM job_categories
WHERE id = $1 LIMIT 1;

-- name: ListJobCategories :many
SELECT * FROM job_categories
ORDER BY name
LIMIT $1
OFFSET $2;

-- name: UpdateJobCategory :one
UPDATE job_categories
SET name = $2, description = $3
WHERE id = $1
RETURNING *;

-- name: DeleteJobCategory :exec
DELETE FROM job_categories
WHERE id = $1;

-- Jobs Queries
-- name: CreateJob :one
INSERT INTO jobs (id, title, description, role, category_id, requirements, location, job_type, status, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetJobById :one
SELECT * FROM jobs
WHERE id = $1 LIMIT 1;

-- name: ListJobs :many
SELECT * FROM jobs
ORDER BY created_at DESC
LIMIT $1
OFFSET $2;

-- name: ListJobsByCategory :many
SELECT * FROM jobs
WHERE category_id = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: ListJobsByStatus :many
SELECT * FROM jobs
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: UpdateJobStatus :one
UPDATE jobs
SET status = $2
WHERE id = $1
RETURNING *;

-- name: UpdateJob :one
UPDATE jobs
SET title = $2, description = $3, role = $4, category_id = $5, requirements = $6, location = $7, job_type = $8, status = $9
WHERE id = $1
RETURNING *;

-- name: DeleteJob :exec
DELETE FROM jobs
WHERE id = $1;

-- Applicants Queries
-- name: CreateApplicant :one
INSERT INTO applicants (id, first_name, last_name, email, github_link, linkedin_link, resume_pdf)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetApplicantById :one
SELECT * FROM applicants
WHERE id = $1 LIMIT 1;

-- name: GetApplicantByEmail :one
SELECT * FROM applicants
WHERE email = $1 LIMIT 1;

-- name: ListApplicants :many
SELECT * FROM applicants
ORDER BY created_at DESC
LIMIT $1
OFFSET $2;

-- name: UpdateApplicant :one
UPDATE applicants
SET first_name = $2, last_name = $3, github_link = $4, linkedin_link = $5, resume_pdf = $6
WHERE id = $1
RETURNING *;

-- name: DeleteApplicant :exec
DELETE FROM applicants
WHERE id = $1;

-- Application Queries
-- name: CreateApplication :one
INSERT INTO application (id, applicant_id, job_id, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CheckApplicationExists :one
SELECT COUNT(*) FROM application
WHERE applicant_id = $1 AND job_id = $2;

-- name: GetApplicationById :one
SELECT * FROM application
WHERE id = $1 LIMIT 1;

-- name: ListApplications :many
SELECT * FROM application
ORDER BY applied_at DESC
LIMIT $1
OFFSET $2;

-- name: ListApplicationsByJob :many
SELECT * FROM application
WHERE job_id = $1
ORDER BY applied_at DESC
LIMIT $2
OFFSET $3;

-- name: ListApplicationsByApplicant :many
SELECT * FROM application
WHERE applicant_id = $1
ORDER BY applied_at DESC
LIMIT $2
OFFSET $3;

-- name: ListApplicationsByStatus :many
SELECT * FROM application
WHERE status = $1
ORDER BY applied_at DESC
LIMIT $2
OFFSET $3;

-- name: UpdateApplicationStatus :one
UPDATE application
SET status = $2, generated_quiz = $3
WHERE id = $1
RETURNING *;

-- name: DeleteApplication :exec
DELETE FROM application
WHERE id = $1;

-- Quiz Queries
-- name: CreateQuiz :one
INSERT INTO quiz (id, application_id, questions)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetQuizById :one
SELECT * FROM quiz
WHERE id = $1 LIMIT 1;

-- name: GetQuizByApplicationId :one
SELECT * FROM quiz
WHERE application_id = $1 LIMIT 1;

-- name: ListQuizzes :many
SELECT * FROM quiz
ORDER BY created_at DESC
LIMIT $1
OFFSET $2;

-- name: UpdateQuiz :one
UPDATE quiz
SET questions = $2
WHERE id = $1
RETURNING *;

-- name: DeleteQuiz :exec
DELETE FROM quiz
WHERE id = $1;

-- Repository Queries
-- name: CreateRepository :one
INSERT INTO repositories (
    id, user_id, github_repo_id, name, full_name, description,
    url, html_url, language, languages, readme_content, readme_html,
    tech_stack, stars, forks, is_private
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING *;

-- name: GetRepositoryByGithubID :one
SELECT * FROM repositories WHERE github_repo_id = $1 LIMIT 1;

-- name: GetRepositoriesByUserId :many
SELECT * FROM repositories 
WHERE user_id = $1 
ORDER BY updated_at DESC;

-- name: UpdateRepository :one
UPDATE repositories
SET 
    description = COALESCE($2, description),
    language = COALESCE($3, language),
    languages = COALESCE($4, languages),
    readme_content = COALESCE($5, readme_content),
    readme_html = COALESCE($6, readme_html),
    tech_stack = COALESCE($7, tech_stack),
    stars = COALESCE($8, stars),
    forks = COALESCE($9, forks),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteRepository :exec
DELETE FROM repositories WHERE id = $1;

-- name: DeleteRepositoriesByUserId :exec
DELETE FROM repositories WHERE user_id = $1;

-- Talent Pool Queries
-- name: SearchTalents :many
SELECT 
    u.id,
    u.first_name,
    u.last_name,
    u.email,
    u.github_username,
    u.avatar_url,
    u.bio,
    u.location,
    u.tech_stack,
    u.experience_level,
    u.availability_status,
    u.talent_status,
    COUNT(DISTINCT r.id) as repositories_count,
    COALESCE(SUM(r.stars), 0) as total_stars,
    COUNT(DISTINCT a.id) as total_applications,
    COUNT(DISTINCT CASE WHEN a.status = 'Accepted' THEN a.id END) as accepted_jobs
FROM users u
LEFT JOIN repositories r ON r.user_id = u.id
LEFT JOIN application a ON a.applicant_id = u.id
WHERE u.role = 'applicant'
    AND ($1::text IS NULL OR u.talent_status = $1)
    AND ($2::text IS NULL OR u.availability_status = $2)
    AND ($3::text IS NULL OR u.experience_level = $3)
    AND ($4::text IS NULL OR u.location ILIKE '%' || $4 || '%')
    AND ($5::jsonb IS NULL OR u.tech_stack @> $5)
GROUP BY u.id
ORDER BY total_stars DESC, repositories_count DESC
LIMIT $6
OFFSET $7;

-- name: GetTalentById :one
SELECT 
    u.*,
    COUNT(DISTINCT r.id) as repositories_count,
    COALESCE(SUM(r.stars), 0) as total_stars
FROM users u
LEFT JOIN repositories r ON r.user_id = u.id
WHERE u.id = $1 AND u.role = 'applicant'
GROUP BY u.id;

-- name: GetTalentApplications :many
SELECT 
    a.*,
    j.title as job_title,
    j.role as job_role
FROM application a
JOIN jobs j ON j.id = a.job_id
WHERE a.applicant_id = $1
ORDER BY a.applied_at DESC;

-- name: GetMatchingTalentsForJob :many
SELECT DISTINCT
    u.id,
    u.first_name,
    u.last_name,
    u.email,
    u.github_username,
    u.avatar_url,
    u.bio,
    u.location,
    u.tech_stack,
    u.experience_level,
    u.availability_status,
    u.talent_status,
    COUNT(DISTINCT r.id) as repositories_count,
    COALESCE(SUM(r.stars), 0) as total_stars
FROM users u
LEFT JOIN repositories r ON r.user_id = u.id
WHERE u.role = 'applicant'
    AND u.availability_status IN ('Available', 'Open to Opportunities')
    AND u.talent_status = 'Active'
    AND u.tech_stack IS NOT NULL
GROUP BY u.id, u.first_name, u.last_name, u.email, u.github_username, u.avatar_url, u.bio, u.location, u.tech_stack, u.experience_level, u.availability_status, u.talent_status
HAVING u.tech_stack ?| (
    SELECT ARRAY(SELECT jsonb_object_keys(tech_stack) FROM jobs WHERE jobs.id = $1)
)
ORDER BY total_stars DESC
LIMIT $2
OFFSET $3;

-- name: UpdateTalentStatus :one
UPDATE users
SET 
    talent_status = COALESCE($2, talent_status),
    availability_status = COALESCE($3, availability_status),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CreateTalentPoolEntry :one
INSERT INTO talent_pool (
    id, user_id, status, tags, notes, rating
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateTalentPoolEntry :one
UPDATE talent_pool
SET 
    status = COALESCE($2, status),
    tags = COALESCE($3, tags),
    notes = COALESCE($4, notes),
    rating = COALESCE($5, rating),
    last_contacted_at = COALESCE($6, last_contacted_at),
    updated_at = now()
WHERE user_id = $1
RETURNING *;

-- name: GetTalentPoolEntry :one
SELECT * FROM talent_pool WHERE user_id = $1 LIMIT 1;