-- Create audit log entry

-- name: CreateAuditLog :one
INSERT INTO audit_logs (
    user_id, user_email, user_role, action,
    entity_type, entity_id, ip_address, user_agent,
    details, old_values, new_values, status, error_message
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- Query audit logs

-- name: GetAuditLogByID :one
SELECT * FROM audit_logs WHERE id = $1;

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
WHERE performed_at BETWEEN $1 AND $2
ORDER BY performed_at DESC
LIMIT $3 OFFSET $4;

-- name: ListAuditLogsByUser :many
SELECT * FROM audit_logs
WHERE user_id = $1
ORDER BY performed_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditLogsByAction :many
SELECT * FROM audit_logs
WHERE action = $1
ORDER BY performed_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditLogsByEntity :many
SELECT * FROM audit_logs
WHERE entity_type = $1 AND entity_id = $2
ORDER BY performed_at DESC;

-- name: ListAuditLogsByIP :many
SELECT * FROM audit_logs
WHERE ip_address = $1
ORDER BY performed_at DESC
LIMIT $2 OFFSET $3;

-- name: GetUserAuditTrail :many
SELECT * FROM audit_logs
WHERE user_id = $1
AND performed_at > NOW() - interval '90 days'
ORDER BY performed_at DESC;

-- name: GetEntityHistory :many
SELECT * FROM audit_logs
WHERE entity_type = $1 AND entity_id = $2
ORDER BY performed_at;

-- Admin audit trail

-- name: CreateAdminAuditTrail :one
INSERT INTO admin_audit_trail (
    audit_log_id, admin_id, admin_notes,
    approval_required, approved_by, approved_at,
    sensitivity_level
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetAdminActions :many
SELECT 
    a.*,
    at.admin_notes,
    at.sensitivity_level,
    at.approval_required,
    at.approved_at
FROM audit_logs a
JOIN admin_audit_trail at ON a.id = at.audit_log_id
WHERE a.performed_at BETWEEN $1 AND $2
ORDER BY a.performed_at DESC
LIMIT $3 OFFSET $4;

-- name: GetPendingApprovals :many
SELECT * FROM admin_audit_trail
WHERE approval_required = true AND approved_at IS NULL
ORDER BY created_at;

-- Compliance and GDPR

-- name: MarkForDeletion :exec
UPDATE audit_logs
SET marked_for_deletion = true
WHERE performed_at < NOW() - (interval '1 day' * data_retention_days);

-- name: GetLogsForExport :many
SELECT * FROM audit_logs
WHERE performed_at BETWEEN $1 AND $2
AND (user_id = $3 OR $3 IS NULL)
ORDER BY performed_at;

-- name: CreateAuditExport :one
INSERT INTO audit_exports (
    exported_by, export_type, date_range_start,
    date_range_end, filter_criteria, file_path
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: IncrementExportDownload :exec
UPDATE audit_exports
SET downloaded_count = downloaded_count + 1
WHERE id = $1;

-- Statistics

-- name: GetAuditStats :many
SELECT 
    DATE(performed_at) as date,
    action,
    COUNT(*) as count
FROM audit_logs
WHERE performed_at BETWEEN $1 AND $2
GROUP BY DATE(performed_at), action
ORDER BY date DESC;

-- name: GetUserActivitySummary :one
SELECT 
    COUNT(*) as total_actions,
    COUNT(DISTINCT DATE(performed_at)) as active_days,
    MAX(performed_at) as last_active,
    MIN(performed_at) as first_active
FROM audit_logs
WHERE user_id = $1;

-- Cleanup

-- name: DeleteMarkedLogs :exec
DELETE FROM audit_logs
WHERE marked_for_deletion = true
AND performed_at < NOW() - interval '30 days';

-- name: ArchiveOldLogs :exec
INSERT INTO audit_logs_archive
SELECT * FROM audit_logs
WHERE performed_at < NOW() - interval '1 year';