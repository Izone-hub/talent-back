package models

import (
	"time"

	"github.com/google/uuid"
)

type QuizAnswerHistory struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	QuizAnswerID   uuid.UUID  `json:"quiz_answer_id" db:"quiz_answer_id"`
	UserAnswer     *string    `json:"user_answer,omitempty" db:"user_answer"`
	TimeSpentSeconds *int     `json:"time_spent_seconds,omitempty" db:"time_spent_seconds"`
	SavedAt        time.Time  `json:"saved_at" db:"saved_at"`
	SaveReason     *string    `json:"save_reason,omitempty" db:"save_reason"`
	IPAddress      *string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent      *string    `json:"user_agent,omitempty" db:"user_agent"`
}
