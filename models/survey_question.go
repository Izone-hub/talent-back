package models

import (
	"time"

	"github.com/google/uuid"
)

// SurveyQuestion represents a yes/no screening question attached to a job.
type SurveyQuestion struct {
	ID             uuid.UUID `json:"id"`
	JobID          uuid.UUID `json:"job_id"`
	QuestionText   string    `json:"question_text"`
	ExpectedAnswer bool      `json:"expected_answer"`
	SortOrder      int       `json:"sort_order"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SurveyQuestionRequest is the payload an admin sends when creating/updating
// screening questions for a job. Each entry is a simple yes/no question.
type SurveyQuestionRequest struct {
	QuestionText   string `json:"question_text"`
	ExpectedAnswer bool   `json:"expected_answer"`
}

// UpsertSurveyQuestionsRequest wraps a list of questions to replace all
// screening questions for a given job in one shot.
type UpsertSurveyQuestionsRequest struct {
	Questions []SurveyQuestionRequest `json:"questions"`
}
