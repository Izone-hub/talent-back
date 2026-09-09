-- name: GetQuizResultFeedback :one
SELECT * FROM quiz_result_feedback
WHERE user_id = $1 AND quiz_attempt_id = $2;

-- name: UpsertQuizResultFeedback :one
INSERT INTO quiz_result_feedback (user_id, quiz_attempt_id, rating, comment)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, quiz_attempt_id)
DO UPDATE SET
    rating = EXCLUDED.rating,
    comment = EXCLUDED.comment,
    updated_at = NOW()
RETURNING *;

-- name: DeleteQuizResultFeedback :exec
DELETE FROM quiz_result_feedback
WHERE user_id = $1 AND quiz_attempt_id = $2;

-- name: ListQuizResultFeedbackByAttempt :many
SELECT * FROM quiz_result_feedback
WHERE quiz_attempt_id = $1
ORDER BY created_at DESC;
