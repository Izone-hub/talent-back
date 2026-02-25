package models

import (
	"time"

	"github.com/google/uuid"
)

type CvVersion struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	FileName  string     `json:"file_name" db:"file_name"`
	FilePath  string     `json:"file_path" db:"file_path"`
	FileSize  int        `json:"file_size" db:"file_size"`
	FileHash  string     `json:"file_hash" db:"file_hash"`
	Version   int        `json:"version" db:"version"`
	IsCurrent bool       `json:"is_current" db:"is_current"`
	UploadedAt time.Time `json:"uploaded_at" db:"uploaded_at"`
	UploadedFromIP *string `json:"uploaded_from_ip,omitempty" db:"uploaded_from_ip"`
	ApplicationID *uuid.UUID `json:"application_id,omitempty" db:"application_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// CvApplicationUsage tracks CV usage across applications.
type CvApplicationUsage struct {
	CvID          uuid.UUID `json:"cv_id" db:"cv_id"`
	ApplicationID uuid.UUID `json:"application_id" db:"application_id"`
	UsedAt        time.Time `json:"used_at" db:"used_at"`
}
