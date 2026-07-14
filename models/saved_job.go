package models

import (
	"time"

	"github.com/google/uuid"
)

type SavedJob struct {
	UserID  uuid.UUID `json:"user_id" db:"user_id"`
	JobID   uuid.UUID `json:"job_id" db:"job_id"`
	SavedAt time.Time `json:"saved_at" db:"saved_at"`
	Notes   *string   `json:"notes,omitempty" db:"notes"`
}
