-- name: GetQuestionFeedback :one
SELECT feedback
FROM question_feedback
WHERE user_id = $1 AND question_id = $2;

-- name: UpsertQuestionFeedback :one
INSERT INTO question_feedback (user_id, question_id, feedback)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, question_id)
DO UPDATE SET feedback = EXCLUDED.feedback, updated_at = NOW()
RETURNING feedback;
