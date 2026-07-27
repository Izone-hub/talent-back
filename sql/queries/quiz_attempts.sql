-- Quiz attempt management

-- name: CreateQuizAttempt :one
INSERT INTO quiz_attempts (
    application_id, user_id, job_id, total_questions,
    questions_per_quiz, time_limit_minutes, passing_score
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetQuizAttemptByID :one
SELECT * FROM quiz_attempts WHERE id = $1;

-- name: GetQuizAttemptByApplication :one
SELECT * FROM quiz_attempts
WHERE application_id = $1;

-- name: GetActiveQuizAttempt :one
SELECT * FROM quiz_attempts
WHERE user_id = $1 
AND status IN ('started', 'in_progress', 'paused')
ORDER BY last_activity_at DESC
LIMIT 1;

-- Status updates

-- name: StartQuiz :one
UPDATE quiz_attempts
SET 
    status = 'in_progress',
    last_activity_at = NOW()
WHERE id = $1 AND status = 'started'
RETURNING *;

-- name: PauseQuiz :one
UPDATE quiz_attempts
SET 
    status = 'paused',
    last_activity_at = NOW()
WHERE id = $1 AND status = 'in_progress'
RETURNING *;

-- name: ResumeQuiz :one
UPDATE quiz_attempts
SET 
    status = 'in_progress',
    last_activity_at = NOW()
WHERE id = $1 AND status = 'paused'
RETURNING *;

-- name: AutoSaveQuiz :exec
UPDATE quiz_attempts
SET 
    last_activity_at = NOW(),
    time_spent_seconds = time_spent_seconds + $2
WHERE id = $1;

-- name: CompleteQuiz :one
UPDATE quiz_attempts
SET 
    status = 'completed',
    completed_at = NOW(),
    score = $2,
    correct_answers = $3,
    incorrect_answers = $4,
    skipped_questions = $5,
    passed = $6,
    updated_at = NOW()
WHERE id = $1 AND status IN ('in_progress', 'paused')
RETURNING *;

-- name: TimeoutQuiz :one
UPDATE quiz_attempts
SET 
    status = 'timed_out',
    completed_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND status = 'in_progress'
RETURNING *;

-- Quiz answers

-- name: SaveAnswer :one
INSERT INTO quiz_answers (
    quiz_attempt_id, question_id, user_answer, is_correct,
    time_spent_seconds, is_skipped
) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (quiz_attempt_id, question_id) 
DO UPDATE SET
    user_answer = EXCLUDED.user_answer,
    is_correct = EXCLUDED.is_correct,
    time_spent_seconds = quiz_answers.time_spent_seconds + EXCLUDED.time_spent_seconds,
    is_skipped = EXCLUDED.is_skipped,
    save_count = quiz_answers.save_count + 1,
    last_saved_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetAnswer :one
SELECT * FROM quiz_answers
WHERE quiz_attempt_id = $1 AND question_id = $2;

-- name: ListAnswers :many
SELECT * FROM quiz_answers
WHERE quiz_attempt_id = $1
ORDER BY created_at;

-- name: GetAnsweredQuestions :many
SELECT question_id, is_correct, time_spent_seconds
FROM quiz_answers
WHERE quiz_attempt_id = $1;

-- name: MarkForReview :exec
UPDATE quiz_answers
SET is_reviewed = true
WHERE quiz_attempt_id = $1 AND question_id = $2;

-- Answer history

-- name: SaveAnswerHistory :exec
INSERT INTO quiz_answer_history (
    quiz_answer_id, user_answer, time_spent_seconds,
    save_reason, ip_address, user_agent
) VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetAnswerHistory :many
SELECT * FROM quiz_answer_history
WHERE quiz_answer_id = $1
ORDER BY saved_at DESC;

-- Quiz results

-- name: CreateQuizResults :one
INSERT INTO quiz_results (
    quiz_attempt_id, easy_correct, easy_total,
    medium_correct, medium_total,
    hard_correct, hard_total,
    expert_correct, expert_total,
    multiple_choice_correct, multiple_choice_total,
    coding_correct, coding_total,
    avg_time_per_question_seconds,
    fastest_question_time_seconds,
    slowest_question_time_seconds,
    ai_feedback, strengths, weaknesses
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
RETURNING *;

-- name: GetQuizResults :one
SELECT * FROM quiz_results
WHERE quiz_attempt_id = $1;

-- name: UpdateAIFeedback :exec
UPDATE quiz_results
SET 
    ai_feedback = $2,
    strengths = $3,
    weaknesses = $4
WHERE quiz_attempt_id = $1;

-- Statistics

-- name: GetUserQuizStats :many
SELECT 
    qa.*,
    j.title as job_title,
    j.company as job_company,
    qa.score,
    qa.passed,
    qr.strengths,
    qr.weaknesses
FROM quiz_attempts qa
JOIN jobs j ON qa.job_id = j.id
LEFT JOIN quiz_results qr ON qa.id = qr.quiz_attempt_id
WHERE qa.user_id = $1 AND qa.status = 'completed'
ORDER BY qa.completed_at DESC
LIMIT $2 OFFSET $3;

-- name: GetQuizAnalytics :many
SELECT 
    DATE(completed_at) as date,
    COUNT(*) as total_attempts,
    AVG(score) as avg_score,
    COUNT(CASE WHEN passed THEN 1 END) as passed_count
FROM quiz_attempts
WHERE completed_at BETWEEN $1 AND $2
GROUP BY DATE(completed_at)
ORDER BY date;

-- name: GetUserQuizAnswers :many
SELECT 
    qa.*
FROM quiz_answers qa
JOIN quiz_attempts qat ON qa.quiz_attempt_id = qat.id
WHERE qat.user_id = $1
ORDER BY qa.created_at DESC;