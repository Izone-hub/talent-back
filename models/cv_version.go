package models

import (
	"time"

	"github.com/google/uuid"
)

type CvVersion struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	FileName       string     `json:"file_name" db:"file_name"`
	FilePath       string     `json:"-" db:"file_path"` // never expose internal path
	FileSize       int        `json:"file_size" db:"file_size"`
	FileHash       string     `json:"file_hash" db:"file_hash"`
	Version        int        `json:"version" db:"version"`
	IsCurrent      bool       `json:"is_current" db:"is_current"`
	UploadedAt     time.Time  `json:"uploaded_at" db:"uploaded_at"`
	UploadedFromIP *string    `json:"uploaded_from_ip,omitempty" db:"uploaded_from_ip"`
	ApplicationID  *uuid.UUID `json:"application_id,omitempty" db:"application_id"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// CvApplicationUsage tracks CV usage across applications.
type CvApplicationUsage struct {
	CvID          uuid.UUID `json:"cv_id" db:"cv_id"`
	ApplicationID uuid.UUID `json:"application_id" db:"application_id"`
	UsedAt        time.Time `json:"used_at" db:"used_at"`
}

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

// CvUploadResponse is the clean API response after a successful CV upload.
type CvUploadResponse struct {
	ID        uuid.UUID `json:"id"`
	FileName  string    `json:"file_name"`
	FileSize  int       `json:"file_size"`
	FileHash  string    `json:"file_hash"`
	Version   int       `json:"version"`
	IsCurrent bool      `json:"is_current"`
	CreatedAt time.Time `json:"created_at"`
}

// CvVersionListResponse wraps a paginated list of CV versions.
type CvVersionListResponse struct {
	Versions []CvUploadResponse `json:"versions"`
	Total    int                `json:"total"`
}

// ToResponse converts a CvVersion model to a clean API response.
func (cv *CvVersion) ToResponse() CvUploadResponse {
	return CvUploadResponse{
		ID:        cv.ID,
		FileName:  cv.FileName,
		FileSize:  cv.FileSize,
		FileHash:  cv.FileHash,
		Version:   cv.Version,
		IsCurrent: cv.IsCurrent,
		CreatedAt: cv.CreatedAt,
	}
}
