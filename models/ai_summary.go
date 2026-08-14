package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AiSummary struct {
	ID         uuid.UUID        `json:"id"`
	UserID     uuid.UUID        `json:"user_id"`
	Summary    json.RawMessage  `json:"summary,omitempty"`
	Strengths  *string          `json:"strengths,omitempty"`
	Weaknesses *string          `json:"weaknesses,omitempty"`
	Model      *string          `json:"model,omitempty"`
	CreatedAt  *time.Time       `json:"created_at,omitempty"`
	CvVersion  *int             `json:"cv_version,omitempty"`
}
