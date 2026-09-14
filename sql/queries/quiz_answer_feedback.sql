-- name: GetQuizAnswerFeedback :one
SELECT * FROM quiz_answer_feedback
WHERE user_id = $1 AND quiz_attempt_id = $2 AND question_id = $3;

-- name: GetQuizAnswerFeedbackByAttempt :many
SELECT * FROM quiz_answer_feedback
WHERE user_id = $1 AND quiz_attempt_id = $2
ORDER BY created_at ASC;

-- name: UpsertQuizAnswerFeedback :one
INSERT INTO quiz_answer_feedback (user_id, quiz_attempt_id, question_id, application_id, feedback)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, quiz_attempt_id, question_id)
DO UPDATE SET
    feedback = EXCLUDED.feedback,
    updated_at = NOW()
RETURNING *;

-- name: DeleteQuizAnswerFeedback :exec
DELETE FROM quiz_answer_feedback
WHERE user_id = $1 AND quiz_attempt_id = $2 AND question_id = $3;

-- name: DeleteQuizAnswerFeedbackByAttempt :exec
DELETE FROM quiz_answer_feedback
WHERE user_id = $1 AND quiz_attempt_id = $2;

-- name: ListQuizAnswerFeedbackByApplication :many
SELECT * FROM quiz_answer_feedback
WHERE application_id = $1
ORDER BY created_at ASC;
