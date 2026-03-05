-- name: CreateTag :one
INSERT INTO tags (name, category, description, color)
VALUES ($1, $2, $3, $4)
ON CONFLICT (name) DO UPDATE SET
    category = EXCLUDED.category,
    description = EXCLUDED.description,
    color = EXCLUDED.color
RETURNING *;

-- name: GetTagByID :one
SELECT * FROM tags WHERE id = $1;

-- name: GetTagByName :one
SELECT * FROM tags WHERE name = $1;

-- name: ListTags :many
SELECT * FROM tags
ORDER BY category, name
LIMIT $1 OFFSET $2;

-- name: ListTagsByCategory :many
SELECT * FROM tags
WHERE category = $1
ORDER BY name;

-- Job-Tag assignments
-- name: AssignTagToJob :exec
INSERT INTO job_tags (job_id, tag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveTagFromJob :exec
DELETE FROM job_tags
WHERE job_id = $1 AND tag_id = $2;

-- name: GetJobTags :many
SELECT t.*
FROM tags t
JOIN job_tags jt ON t.id = jt.tag_id
WHERE jt.job_id = $1
ORDER BY t.category, t.name;

-- name: GetJobsByTag :many
SELECT j.*
FROM jobs j
JOIN job_tags jt ON j.id = jt.job_id
WHERE jt.tag_id = $1 AND j.status = 'published'
ORDER BY j.published_at DESC
LIMIT $2 OFFSET $3;