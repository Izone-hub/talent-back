-- Save/Unsave operations

-- name: SaveJob :exec
INSERT INTO saved_jobs (user_id, job_id, notes)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, job_id) 
DO UPDATE SET 
    notes = EXCLUDED.notes,
    saved_at = NOW();

-- name: UnsaveJob :exec
DELETE FROM saved_jobs
WHERE user_id = $1 AND job_id = $2;

-- name: GetSavedJobsByUser :many
SELECT 
    sj.*,
    j.title,
    j.company,
    j.location,
    j.job_type,
    j.salary_min,
    j.salary_max,
    j.salary_currency,
    j.remote_possible,
    j.status as job_status
FROM saved_jobs sj
JOIN jobs j ON sj.job_id = j.id
WHERE sj.user_id = $1
ORDER BY sj.saved_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSavedJobsByUserBasic :many
SELECT sj.* FROM saved_jobs sj
WHERE sj.user_id = $1
ORDER BY sj.saved_at DESC;

-- name: IsJobSaved :one
SELECT EXISTS(
    SELECT 1 FROM saved_jobs
    WHERE user_id = $1 AND job_id = $2
) AS is_saved;

-- name: GetSavedJobNotes :one
SELECT notes FROM saved_jobs
WHERE user_id = $1 AND job_id = $2;

-- name: UpdateSavedJobNotes :exec
UPDATE saved_jobs
SET notes = $3, saved_at = NOW()
WHERE user_id = $1 AND job_id = $2;

-- name: CountSavedJobsByUser :one
SELECT COUNT(*) FROM saved_jobs
WHERE user_id = $1;

-- name: GetMostSavedJobs :many
SELECT 
    j.id,
    j.title,
    j.company,
    COUNT(sj.user_id) as save_count
FROM jobs j
LEFT JOIN saved_jobs sj ON j.id = sj.job_id
WHERE j.status = 'published'
GROUP BY j.id
ORDER BY save_count DESC
LIMIT $1;

-- name: DeleteAllSavedJobs :exec
DELETE FROM saved_jobs
WHERE user_id = $1;