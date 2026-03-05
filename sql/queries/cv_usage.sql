-- Track which CVs are used for which applications

-- name: LinkCVToApplication :exec
INSERT INTO cv_application_usage (cv_id, application_id)
VALUES ($1, $2)
ON CONFLICT (cv_id, application_id) DO NOTHING;

-- name: UnlinkCVFromApplication :exec
DELETE FROM cv_application_usage
WHERE cv_id = $1 AND application_id = $2;

-- name: GetApplicationsByCV :many
SELECT 
    a.*,
    j.title,
    j.company
FROM cv_application_usage cu
JOIN job_applications a ON cu.application_id = a.id
JOIN jobs j ON a.job_id = j.id
WHERE cu.cv_id = $1
ORDER BY cu.used_at DESC;

-- name: GetCVsForApplication :many
SELECT c.*
FROM cv_application_usage cu
JOIN cv_versions c ON cu.cv_id = c.id
WHERE cu.application_id = $1
ORDER BY cu.used_at DESC;

-- name: GetLatestCVForApplication :one
SELECT c.*
FROM cv_application_usage cu
JOIN cv_versions c ON cu.cv_id = c.id
WHERE cu.application_id = $1
ORDER BY cu.used_at DESC
LIMIT 1;

-- name: CountCVUsage :one
SELECT COUNT(*) FROM cv_application_usage
WHERE cv_id = $1;

-- name: GetCVUsageHistory :many
SELECT 
    cu.*,
    a.status as application_status,
    j.title as job_title,
    j.company as job_company
FROM cv_application_usage cu
JOIN job_applications a ON cu.application_id = a.id
JOIN jobs j ON a.job_id = j.id
WHERE cu.cv_id = $1
ORDER BY cu.used_at DESC;

-- name: RemoveCVFromAllApplications :exec
DELETE FROM cv_application_usage
WHERE cv_id = $1;

-- name: TransferCVUsage :exec
UPDATE cv_application_usage
SET cv_id = $2
WHERE cv_id = $1;

-- name: GetApplicationsUsingCV :many
SELECT DISTINCT application_id
FROM cv_application_usage
WHERE cv_id = $1;