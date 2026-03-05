-- name: CreateOrUpdateUser :one
INSERT INTO users (
    github_id, 
    github_username, 
    email, 
    avatar_url, 
    name, 
    github_access_token, 
    github_token_expires_at,
    last_login_at,
    public_repos,
    public_gists,
    followers,
    following,
    hireable,
    blog,
    company,
    location,
    bio,
    twitter_username,
    top_languages,
    contribution_count
)
VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
ON CONFLICT (github_id) 
DO UPDATE SET 
    github_username = EXCLUDED.github_username,
    email = EXCLUDED.email,
    avatar_url = EXCLUDED.avatar_url,
    name = EXCLUDED.name,
    github_access_token = EXCLUDED.github_access_token,
    github_token_expires_at = EXCLUDED.github_token_expires_at,
    last_login_at = NOW(),
    updated_at = NOW(),
    public_repos = EXCLUDED.public_repos,
    public_gists = EXCLUDED.public_gists,
    followers = EXCLUDED.followers,
    following = EXCLUDED.following,
    hireable = EXCLUDED.hireable,
    blog = EXCLUDED.blog,
    company = EXCLUDED.company,
    location = EXCLUDED.location,
    bio = EXCLUDED.bio,
    twitter_username = EXCLUDED.twitter_username,
    top_languages = EXCLUDED.top_languages,
    contribution_count = EXCLUDED.contribution_count
RETURNING *;

-- name: GetUserByGitHubID :one
SELECT * FROM users 
WHERE github_id = $1 LIMIT 1;

-- name: GetUserByGitHubUsername :one
SELECT * FROM users
WHERE github_username = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users 
WHERE id = $1 LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users 
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateUserRole :one
UPDATE users 
SET role = $2, updated_at = NOW()
WHERE github_id = $1
RETURNING *;

-- name: GetAdminUsers :many
SELECT * FROM users 
WHERE role = 'admin'
ORDER BY github_username;

-- name: IsUserAdmin :one
SELECT EXISTS(
    SELECT 1 FROM users 
    WHERE github_id = $1 AND role = 'admin'
) as is_admin;

-- name: DeleteUser :exec
DELETE FROM users 
WHERE github_id = $1;