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
    u.avatar_url as user_avatar_url,
    u.acceptance_job_id as acceptance_job_id
FROM job_applications a
JOIN jobs j ON a.job_id = j.id
JOIN users u ON a.user_id = u.id
WHERE a.id = $1;

-- name: ListMyApplicationsDashboard :many
-- Returns the rows shown on the user's "My Applications" page, pre-joined with
-- the job fields the UI renders, limited to the visible page, and carrying
-- window-function stats (total / active / accepted / rejected) computed over
-- exactly the returned rows so the client performs no aggregation.
--
-- The accepted-job filter reads the canonical users.acceptance_job_id column:
-- when it is non-NULL only the application for that exact job is returned;
-- when it is NULL every application is returned. The "accepted" stat is the
-- count of rows whose job is the user's canonical accepted job; application
-- status stays purely process history and never records acceptance.
SELECT
    page.id,
    page.job_id,
    page.status,
    page.submitted_at,
    page.updated_at,
    page.quiz_id,
    page.quiz_score,
    page.quiz_passed,
    page.employer_feedback,
    page.applicant_email,
    page.job_title,
    page.job_company,
    page.job_location,
    page.job_type,
    page.job_status,
    COUNT(*) OVER () AS total,
    COUNT(*) FILTER (WHERE page.status IN ('submitted', 'quiz_started', 'quiz_completed', 'under_review', 'shortlisted', 'interviewed') AND NOT page.is_accepted_job) OVER () AS active,
    COUNT(*) FILTER (WHERE page.is_accepted_job) OVER () AS accepted,
    COUNT(*) FILTER (WHERE page.status = 'rejected') OVER () AS rejected,
    page.is_accepted_job
FROM (
    SELECT
        a.id,
        a.job_id,
        a.status,
        a.submitted_at,
        a.updated_at,
        a.quiz_id,
        a.quiz_score,
        a.quiz_passed,
        a.employer_feedback,
        a.applicant_email,
        a.created_at,
        j.title AS job_title,
        j.company AS job_company,
        j.location AS job_location,
        j.job_type,
        j.status AS job_status,
        (u.acceptance_job_id IS NOT NULL AND a.job_id = u.acceptance_job_id) AS is_accepted_job
    FROM job_applications a
    JOIN jobs j ON j.id = a.job_id
    JOIN users u ON u.id = a.user_id
    WHERE a.user_id = $1
      AND (u.acceptance_job_id IS NULL OR a.job_id = u.acceptance_job_id)
    ORDER BY a.created_at DESC
    LIMIT $2 OFFSET $3
) page
ORDER BY page.created_at DESC;

-- name: ListApplicationsByUser :many
-- All applications for a given user (admin view), newest first, with the job
-- snapshot fields the admin user detail page renders. Unlike the user's own
-- dashboard query this has no acceptance filter: admins see the full history.
SELECT
    a.id,
    a.job_id,
    a.status,
    a.submitted_at,
    a.updated_at,
    a.quiz_id,
    a.quiz_score,
    a.quiz_passed,
    a.employer_feedback,
    a.applicant_email,
    j.title AS job_title,
    j.company AS job_company,
    j.location AS job_location,
    j.job_type,
    j.status AS job_status,
    (u.acceptance_job_id IS NOT NULL AND a.job_id = u.acceptance_job_id) AS is_accepted_job
FROM job_applications a
JOIN jobs j ON j.id = a.job_id
JOIN users u ON u.id = a.user_id
WHERE a.user_id = $1
ORDER BY a.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListApplicationsByJob :many
SELECT 
    a.*,
    u.github_username,
    u.avatar_url,
    u.email,
    u.name,
    u.acceptance_job_id as acceptance_job_id
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

-- Note: accepting a job does NOT update job_applications.status. Acceptance is
-- a user-level relationship stored only in users.acceptance_job_id (handled in
-- service/application.go AcceptApplication, which sets users.acceptance_job_id
-- via SetUserAcceptanceJob); job_applications.status remains purely
-- application-process history.

-- name: RejectApplication :one
UPDATE job_applications
SET 
    status = 'rejected',
    rejection_reason = $2,
    employer_feedback = $3,
    updated_at = NOW()
WHERE id = $1 AND status <> 'withdrawn'
RETURNING *;

-- name: WithdrawApplication :one
UPDATE job_applications
SET 
    status = 'withdrawn',
    updated_at = NOW()
WHERE id = $1 AND user_id = $2 AND status <> 'rejected'
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

-- name: ListPublishedJobApplicationStats :many
-- Per-job applicant counters for the admin applications overview, computed
-- for ALL published jobs in a single pass (no per-job round trips). The
-- "accepted" counter is the number of applicants whose canonical acceptance
-- (users.acceptance_job_id) points at that job, not an application status.
-- total_applications matches the raw application rows an admin can open for a
-- job (all statuses, including drafts).
SELECT
    j.id AS job_id,
    j.title,
    j.company,
    j.category,
    j.location,
    j.job_type,
    j.remote_possible,
    COUNT(ja.id)::bigint AS total_applications,
    COUNT(CASE WHEN ja.status = 'submitted' THEN 1 END)::bigint AS submitted,
    COUNT(CASE WHEN ja.status = 'quiz_started' THEN 1 END)::bigint AS quiz_started,
    COUNT(CASE WHEN ja.status = 'quiz_completed' THEN 1 END)::bigint AS quiz_completed,
    COUNT(CASE WHEN ja.status = 'under_review' THEN 1 END)::bigint AS under_review,
    COUNT(CASE WHEN ja.status = 'shortlisted' THEN 1 END)::bigint AS shortlisted,
    COUNT(CASE WHEN ja.status = 'interviewed' THEN 1 END)::bigint AS interviewed,
    COUNT(CASE WHEN ja.status = 'rejected' THEN 1 END)::bigint AS rejected,
    COUNT(CASE WHEN ja.status = 'withdrawn' THEN 1 END)::bigint AS withdrawn,
    COUNT(CASE WHEN u.acceptance_job_id = ja.job_id THEN 1 END)::bigint AS accepted
FROM jobs j
LEFT JOIN job_applications ja ON ja.job_id = j.id
LEFT JOIN users u ON u.id = ja.user_id
WHERE j.status = 'published'
GROUP BY j.id, j.title, j.company, j.category, j.location, j.job_type, j.remote_possible
ORDER BY total_applications DESC, j.created_at DESC;

-- name: ListQuizCompletedCandidates :many
-- Quiz-completed applicants across all published jobs, carrying only the
-- fields the admin "Quiz Completed Users" card renders, pre-sorted by score
-- so the client performs no aggregation.
SELECT
    a.id AS application_id,
    a.job_id,
    a.quiz_score,
    COALESCE(u.name, a.applicant_name)::text AS applicant_name,
    COALESCE(u.email, a.applicant_email)::text AS applicant_email,
    COALESCE(u.github_username, a.github_username) AS applicant_github_username,
    COALESCE(u.avatar_url, a.applicant_avatar_url)::text AS applicant_avatar_url,
    j.title AS job_title
FROM job_applications a
JOIN jobs j ON j.id = a.job_id
LEFT JOIN users u ON u.id = a.user_id
WHERE j.status = 'published'
  AND a.quiz_id IS NOT NULL
  AND a.quiz_completed_at IS NOT NULL
  AND a.status NOT IN ('draft', 'quiz_started')
ORDER BY a.quiz_score DESC NULLS LAST, a.quiz_completed_at DESC
LIMIT $1;

-- name: GetApplicationCountsByJob :one
-- Counts applicants per process state. "accepted" is the number of applicants
-- whose canonical acceptance (users.acceptance_job_id) points at this job,
-- not an application status.
SELECT 
    COUNT(*) as total_applications,
    COUNT(CASE WHEN ja.status = 'submitted' THEN 1 END) as submitted,
    COUNT(CASE WHEN ja.status = 'quiz_completed' THEN 1 END) as quiz_completed,
    COUNT(CASE WHEN ja.status = 'under_review' THEN 1 END) as under_review,
    COUNT(CASE WHEN ja.status = 'shortlisted' THEN 1 END) as shortlisted,
    COUNT(CASE WHEN u.acceptance_job_id = ja.job_id THEN 1 END) as accepted,
    COUNT(CASE WHEN ja.status = 'rejected' THEN 1 END) as rejected
FROM job_applications ja
JOIN users u ON u.id = ja.user_id
WHERE ja.job_id = $1;

-- name: ListApplicationCountsByJob :many
-- Per-job applicant counters for the admin applications overview. Computes
-- every count the admin page needs for all published jobs in a single pass:
-- the "accepted" counter is the number of applicants whose canonical
-- acceptance (users.acceptance_job_id) points at that job, not an
-- application status. Draft rows are included in total_applications so the
-- number matches what the admin can open per job.
SELECT
    ja.job_id,
    COUNT(*)::bigint AS total_applications,
    COUNT(CASE WHEN ja.status = 'submitted' THEN 1 END)::bigint AS submitted,
    COUNT(CASE WHEN ja.status = 'quiz_completed' THEN 1 END)::bigint AS quiz_completed,
    COUNT(CASE WHEN ja.status = 'under_review' THEN 1 END)::bigint AS under_review,
    COUNT(CASE WHEN ja.status = 'shortlisted' THEN 1 END)::bigint AS shortlisted,
    COUNT(CASE WHEN u.acceptance_job_id = ja.job_id THEN 1 END)::bigint AS accepted,
    COUNT(CASE WHEN ja.status = 'rejected' THEN 1 END)::bigint AS rejected
FROM job_applications ja
JOIN jobs j ON j.id = ja.job_id AND j.status = 'published'
JOIN users u ON u.id = ja.user_id
GROUP BY ja.job_id
ORDER BY total_applications DESC, ja.job_id;

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