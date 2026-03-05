-- CV CRUD operations

-- name: CreateCV :one
INSERT INTO cv_versions (
    user_id, file_name, file_path, file_size, 
    file_hash, version, is_current, uploaded_from_ip,
    application_id
) VALUES (
    $1, $2, $3, $4, $5, 
    COALESCE((SELECT MAX(version) + 1 FROM cv_versions WHERE user_id = $1), 1),
    true, $6, $7
) RETURNING *;

-- name: GetCVByID :one
SELECT * FROM cv_versions WHERE id = $1;

-- name: GetCVByHash :one
SELECT * FROM cv_versions 
WHERE user_id = $1 AND file_hash = $2 
ORDER BY version DESC LIMIT 1;

-- name: ListCVsByUser :many
SELECT * FROM cv_versions
WHERE user_id = $1
ORDER BY 
    is_current DESC,
    version DESC,
    created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetCurrentCV :one
SELECT * FROM cv_versions
WHERE user_id = $1 AND is_current = true
LIMIT 1;

-- name: GetCVVersions :many
SELECT 
    version,
    file_name,
    file_size,
    created_at,
    is_current,
    application_id
FROM cv_versions
WHERE user_id = $1
ORDER BY version DESC;

-- name: SetCurrentCV :exec
UPDATE cv_versions
SET is_current = false
WHERE user_id = $1 AND is_current = true;

-- name: UpdateCV :one
UPDATE cv_versions
SET 
    file_name = $2,
    file_path = $3,
    file_size = $4,
    file_hash = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCV :exec
DELETE FROM cv_versions
WHERE id = $1 AND user_id = $2;

-- name: DeleteOldCVs :exec
DELETE FROM cv_versions c
WHERE c.user_id = $1
AND c.id NOT IN (
    SELECT id FROM cv_versions
    WHERE user_id = $1
    ORDER BY version DESC
    LIMIT 5  -- Keep last 5 versions
);

-- name: CountUserCVs :one
SELECT COUNT(*) FROM cv_versions
WHERE user_id = $1;

-- name: GetCVsByApplication :many
SELECT 
    c.*,
    cu.used_at
FROM cv_versions c
JOIN cv_application_usage cu ON c.id = cu.cv_id
WHERE cu.application_id = $1
ORDER BY cu.used_at DESC;

-- name: FindDuplicateCV :one
SELECT * FROM cv_versions
WHERE user_id = $1 AND file_hash = $2
LIMIT 1;

-- name: GetCVStats :one
SELECT 
    COUNT(*) as total_cvs,
    SUM(file_size) as total_storage_bytes,
    MAX(version) as latest_version,
    COUNT(DISTINCT application_id) as applications_used
FROM cv_versions
WHERE user_id = $1;