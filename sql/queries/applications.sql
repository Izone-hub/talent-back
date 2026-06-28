-- Application CRUD operations

-- name: CreateApplication :one
INSERT INTO job_applications (
    job_id, user_id, 
    github_username, github_id, applicant_email, applicant_name, applicant_avatar_url,
    cover_letter, proposed_salary, proposed_salary_currency, availability_date,
    portfolio_url, linkedin_url, notes, status, submitted_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
) RETURNING *;

-- name: GetApplicationByID :one
SELECT * FROM job_applications WHERE id = $1;

-- name: GetApplicationByJobAndUser :one
SELECT * FROM job_applications 
WHERE job_id = $1 AND user_id = $2 
LIMIT 1;

-- name: GetApplicationWithDetails :one
SELECT 
    a.*,
    j.title as job_title,
    j.company as job_company,
    j.location as job_location,
    j.job_type,
    u.github_username as user_github_username,
    u.avatar_url as user_avatar_url
FROM job_applications a
JOIN jobs j ON a.job_id = j.id
JOIN users u ON a.user_id = u.id
WHERE a.id = $1;

-- name: ListApplicationsByUser :many
SELECT 
    a.*,
    j.title as job_title,
    j.company as job_company,
    j.status as job_status,
    COALESCE(
        (SELECT json_agg(json_build_object(
            'id', c.id,
            'file_name', c.file_name,
            'version', c.version,
            'uploaded_at', c.uploaded_at
        ))
        FROM cv_application_usage cu
        JOIN cv_versions c ON cu.cv_id = c.id
        WHERE cu.application_id = a.id),
        '[]'::json
    ) as cvs
FROM job_applications a
JOIN jobs j ON a.job_id = j.id
WHERE a.user_id = $1
ORDER BY a.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListApplicationsByJob :many
SELECT 
    a.*,
    u.github_username,
    u.avatar_url,
    u.email,
    u.name
FROM job_applications a
JOIN users u ON a.user_id = u.id
WHERE a.job_id = $1
ORDER BY 
    CASE a.status::text
        WHEN 'draft' THEN 1
        WHEN 'submitted' THEN 2
        WHEN 'quiz_started' THEN 3
        WHEN 'quiz_completed' THEN 4
        WHEN 'under_review' THEN 5
        WHEN 'shortlisted' THEN 6
        WHEN 'interviewed' THEN 7
        WHEN 'accepted' THEN 8
        WHEN 'rejected' THEN 9
        WHEN 'withdrawn' THEN 10
    END,
    a.submitted_at DESC
LIMIT $2 OFFSET $3;

-- name: ListApplicationsByStatus :many
SELECT * FROM job_applications
WHERE status = $1
ORDER BY submitted_at DESC
LIMIT $2 OFFSET $3;

-- Application Status Updates (Lifecycle)

-- name: SubmitApplication :one
UPDATE job_applications
SET 
    status = 'submitted',
    submitted_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND user_id = $2 AND status = 'draft'
RETURNING *;

-- name: ApplicationStartQuiz :one
UPDATE job_applications
SET 
    status = 'quiz_started',
    quiz_id = $2,
    updated_at = NOW()
WHERE id = $1 AND status = 'submitted'
RETURNING *;

-- name: ApplicationCompleteQuiz :one
UPDATE job_applications
SET 
    status = 'quiz_completed',
    quiz_score = $2,
    quiz_completed_at = NOW(),
    quiz_passed = $3,
    updated_at = NOW()
WHERE id = $1 AND status = 'quiz_started'
RETURNING *;

-- name: StartReview :one
UPDATE job_applications
SET 
    status = 'under_review',
    reviewed_at = NOW(),
    reviewed_by = $2,
    updated_at = NOW()
WHERE id = $1 AND status IN ('submitted', 'quiz_completed')
RETURNING *;

-- name: ShortlistApplication :one
UPDATE job_applications
SET 
    status = 'shortlisted',
    updated_at = NOW()
WHERE id = $1 AND status = 'under_review'
RETURNING *;

-- name: MarkInterviewed :one
UPDATE job_applications
SET 
    status = 'interviewed',
    updated_at = NOW()
WHERE id = $1 AND status = 'shortlisted'
RETURNING *;

-- name: AcceptApplication :one
UPDATE job_applications
SET 
    status = 'accepted',
    updated_at = NOW()
WHERE id = $1 AND status IN ('submitted', 'quiz_completed', 'under_review', 'shortlisted', 'interviewed')
RETURNING *;

-- name: RejectApplication :one
UPDATE job_applications
SET 
    status = 'rejected',
    rejection_reason = $2,
    employer_feedback = $3,
    updated_at = NOW()
WHERE id = $1 AND status NOT IN ('accepted', 'withdrawn')
RETURNING *;

-- name: WithdrawApplication :one
UPDATE job_applications
SET 
    status = 'withdrawn',
    updated_at = NOW()
WHERE id = $1 AND user_id = $2 AND status NOT IN ('accepted', 'rejected')
RETURNING *;

-- name: AddEmployerFeedback :one
UPDATE job_applications
SET 
    employer_feedback = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateQuizScore :one
UPDATE job_applications
SET 
    quiz_score = $2,
    quiz_passed = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- Check/Validation Queries

-- name: HasUserApplied :one
SELECT EXISTS(
    SELECT 1 FROM job_applications 
    WHERE job_id = $1 AND user_id = $2
) AS has_applied;

-- name: GetApplicationCountsByJob :one
SELECT 
    COUNT(*) as total_applications,
    COUNT(CASE WHEN status = 'submitted' THEN 1 END) as submitted,
    COUNT(CASE WHEN status = 'quiz_completed' THEN 1 END) as quiz_completed,
    COUNT(CASE WHEN status = 'under_review' THEN 1 END) as under_review,
    COUNT(CASE WHEN status = 'shortlisted' THEN 1 END) as shortlisted,
    COUNT(CASE WHEN status = 'accepted' THEN 1 END) as accepted,
    COUNT(CASE WHEN status = 'rejected' THEN 1 END) as rejected
FROM job_applications
WHERE job_id = $1;

-- name: GetUserApplicationStats :many
SELECT 
    j.company,
    j.title,
    a.status,
    a.submitted_at,
    a.quiz_score,
    a.quiz_passed
FROM job_applications a
JOIN jobs j ON a.job_id = j.id
WHERE a.user_id = $1
ORDER BY a.submitted_at DESC;

-- name: GetRecentApplications :many
SELECT * FROM job_applications
WHERE submitted_at > NOW() - interval '7 days'
ORDER BY submitted_at DESC
LIMIT $1;

-- name: DeleteDraftApplication :exec
DELETE FROM job_applications
WHERE id = $1 AND user_id = $2 AND status = 'draft';