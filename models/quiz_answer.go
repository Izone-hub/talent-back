package models

import (
	"time"

	"github.com/google/uuid"
)

type QuizAnswer struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	QuizAttemptID uuid.UUID  `json:"quiz_attempt_id" db:"quiz_attempt_id"`
	QuestionID    uuid.UUID  `json:"question_id" db:"question_id"`
	UserAnswer    *string    `json:"user_answer,omitempty" db:"user_answer"`
	IsCorrect     *bool      `json:"is_correct,omitempty" db:"is_correct"`
	LastSavedAt   time.Time  `json:"last_saved_at" db:"last_saved_at"`
	SaveCount     int        `json:"save_count" db:"save_count"`
	TimeSpentSeconds int     `json:"time_spent_seconds" db:"time_spent_seconds"`
	CodeOutput    *string    `json:"code_output,omitempty" db:"code_output"`
	ExecutionTimeMs *int     `json:"execution_time_ms,omitempty" db:"execution_time_ms"`
	MemoryUsedMb  *float64   `json:"memory_used_mb,omitempty" db:"memory_used_mb"`
	IsSkipped     bool       `json:"is_skipped" db:"is_skipped"`
	IsReviewed    bool       `json:"is_reviewed" db:"is_reviewed"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}
