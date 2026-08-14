-- name: GetDashboardStats :one
SELECT
    (SELECT COUNT(*) FROM users)::int AS total_users,
    (SELECT COUNT(*) FROM jobs WHERE status = 'published')::int AS active_jobs,
    (SELECT COUNT(*) FROM job_applications WHERE status IN ('submitted', 'quiz_completed', 'under_review'))::int AS pending_applications,
    (SELECT COUNT(*) FROM job_applications WHERE status != 'draft')::int AS total_applications,
    (SELECT COUNT(*) FROM users WHERE created_at >= CURRENT_DATE)::int AS new_users_today,
    (SELECT COUNT(*) FROM job_applications WHERE submitted_at >= CURRENT_DATE)::int AS new_applications_today;

-- name: GetRecentActivity :many
SELECT
    u.github_username,
    u.avatar_url,
    ja.status,
    ja.submitted_at,
    j.title AS job_title
FROM job_applications ja
JOIN users u ON ja.user_id = u.id
JOIN jobs j ON ja.job_id = j.id
WHERE ja.submitted_at IS NOT NULL
ORDER BY ja.submitted_at DESC
LIMIT $1;