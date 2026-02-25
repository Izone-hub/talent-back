-- Bulk operations for admin/reporting

-- name: BulkUpdateApplicationStatus :many
UPDATE job_applications
SET 
    status = $2,
    updated_at = NOW()
WHERE job_id = $1 AND status = $3
RETURNING *;

-- name: ArchiveOldApplications :many
UPDATE job_applications
SET 
    status = 'withdrawn',
    updated_at = NOW()
WHERE submitted_at < NOW() - interval '6 months'
AND status IN ('submitted', 'quiz_started', 'quiz_completed')
RETURNING *;

-- name: GetApplicationsByDateRange :many
SELECT * FROM job_applications
WHERE submitted_at BETWEEN $1 AND $2
ORDER BY submitted_at;

-- name: GetCVsByDateRange :many
SELECT * FROM cv_versions
WHERE created_at BETWEEN $1 AND $2
ORDER BY created_at;

-- name: CleanupUnusedCVs :many
DELETE FROM cv_versions
WHERE id NOT IN (
    SELECT DISTINCT cv_id FROM cv_application_usage
)
AND created_at < NOW() - interval '30 days'
RETURNING *;