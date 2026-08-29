-- name: CreateSurveyQuestion :one
INSERT INTO job_survey_questions (job_id, question_text, expected_answer, sort_order)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetSurveyQuestionsByJob :many
SELECT * FROM job_survey_questions
WHERE job_id = $1
ORDER BY sort_order ASC, created_at ASC;

-- name: DeleteSurveyQuestionsByJob :exec
DELETE FROM job_survey_questions WHERE job_id = $1;

-- name: DeleteSurveyQuestion :exec
DELETE FROM job_survey_questions WHERE id = $1 AND job_id = $2;
