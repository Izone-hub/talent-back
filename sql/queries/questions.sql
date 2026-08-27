-- Question CRUD

-- name: CreateQuestion :one
INSERT INTO questions (
    question_text, question_type, difficulty, options, correct_answer,
    explanation, time_limit_seconds, points, tags, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetQuestionByID :one
SELECT * FROM questions WHERE id = $1;

-- name: ListQuestions :many
SELECT * FROM questions
WHERE is_active = true
  AND ($3::text = '' OR
    to_tsvector('english', question_text) @@ plainto_tsquery('english', $3) OR
    EXISTS (SELECT 1 FROM unnest(tags) t WHERE lower(t) LIKE '%' || lower($3) || '%'))
ORDER BY
  CASE WHEN $3::text != '' THEN
    ts_rank(to_tsvector('english', question_text), plainto_tsquery('english', $3))
  ELSE 0 END DESC,
  difficulty, created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountActiveQuestions :one
SELECT COUNT(*) FROM questions
WHERE is_active = true
  AND ($1::text = '' OR
    to_tsvector('english', question_text) @@ plainto_tsquery('english', $1) OR
    EXISTS (SELECT 1 FROM unnest(tags) t WHERE lower(t) LIKE '%' || lower($1) || '%'));

-- name: ListQuestionsByDifficulty :many
SELECT * FROM questions
WHERE difficulty = $1 AND is_active = true
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListQuestionsByTag :many
SELECT q.* FROM questions q
WHERE $1 = ANY(q.tags) AND is_active = true
ORDER BY q.difficulty, q.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetRandomQuestions :many
SELECT * FROM questions
WHERE is_active = true
AND difficulty = $1
ORDER BY RANDOM()
LIMIT $2;

-- name: GetQuestionsForQuiz :many
SELECT * FROM questions
WHERE id = ANY($1::uuid[])
ORDER BY 
    CASE difficulty
        WHEN 'easy' THEN 1
        WHEN 'medium' THEN 2
        WHEN 'hard' THEN 3
        WHEN 'expert' THEN 4
    END;

-- name: UpdateQuestion :one
UPDATE questions
SET 
    question_text = COALESCE($2, question_text),
    options = COALESCE($3, options),
    correct_answer = COALESCE($4, correct_answer),
    explanation = COALESCE($5, explanation),
    time_limit_seconds = COALESCE($6, time_limit_seconds),
    points = COALESCE($7, points),
    tags = COALESCE($8, tags),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteQuestion :exec
UPDATE questions
SET is_active = false, updated_at = NOW()
WHERE id = $1;

-- name: AssignTagToQuestion :exec
INSERT INTO question_tags (question_id, tag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: IncrementQuestionUsage :exec
UPDATE questions
SET usage_count = usage_count + 1
WHERE id = $1;

-- Coding questions

-- name: CreateCodingQuestion :one
INSERT INTO coding_questions (
    question_id, language, code_template, test_cases,
    execution_time_limit, memory_limit
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetCodingQuestion :one
SELECT * FROM coding_questions
WHERE question_id = $1;

-- name: UpdateCodingQuestion :one
UPDATE coding_questions
SET 
    language = COALESCE($2, language),
    code_template = COALESCE($3, code_template),
    test_cases = COALESCE($4, test_cases),
    execution_time_limit = COALESCE($5, execution_time_limit),
    memory_limit = COALESCE($6, memory_limit)
WHERE question_id = $1
RETURNING *;