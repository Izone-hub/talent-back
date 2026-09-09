package models

import (
	"time"

	"github.com/google/uuid"
)

type QuizAttemptQuestion struct {
	ID            uuid.UUID `json:"id" db:"id"`
	QuizAttemptID uuid.UUID `json:"quiz_attempt_id" db:"quiz_attempt_id"`
	QuestionID    uuid.UUID `json:"question_id" db:"question_id"`
	QuestionOrder int       `json:"question_order" db:"question_order"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}
