-- name: CreateAdmin :one
INSERT INTO admins (user_id, github_id, github_login, email, role, permissions)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpsertAdmin :one
INSERT INTO admins (user_id, github_id, github_login, email, role, permissions)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id) DO UPDATE SET
    github_login = EXCLUDED.github_login,
    email = EXCLUDED.email,
    role = EXCLUDED.role,
    permissions = COALESCE(EXCLUDED.permissions, admins.permissions),
    updated_at = NOW()
RETURNING *;

-- name: GetAdminByGitHubID :one
SELECT * FROM admins WHERE github_id = $1 LIMIT 1;

-- name: GetAdminByUserID :one
SELECT * FROM admins WHERE user_id = $1 LIMIT 1;

-- name: ListAdmins :many
SELECT * FROM admins ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: UpdateAdminPermissions :one
UPDATE admins SET permissions = $2, role = $3, updated_at = NOW() WHERE id = $1 RETURNING *;

-- name: DeleteAdmin :exec
DELETE FROM admins WHERE id = $1;

-- name: IsGitHubAllowlisted :one
SELECT EXISTS(
    SELECT 1 FROM admin_allowlist WHERE github_id = $1
) as allowlisted;

-- name: AddAllowlistEntry :one
INSERT INTO admin_allowlist (github_id, github_login, note, added_by) VALUES ($1, $2, $3, $4)
ON CONFLICT (github_id) DO UPDATE SET github_login = EXCLUDED.github_login, note = EXCLUDED.note
RETURNING *;

-- name: RemoveAllowlistEntry :exec
DELETE FROM admin_allowlist WHERE github_id = $1;

-- name: ListAllowlist :many
SELECT * FROM admin_allowlist ORDER BY added_at DESC LIMIT $1 OFFSET $2;
