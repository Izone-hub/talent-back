package models

import (
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the status of a job posting.
type JobStatus string

const (
	JobStatusDraft     JobStatus = "draft"
	JobStatusPublished JobStatus = "published"
	JobStatusClosed    JobStatus = "closed"
	JobStatusArchived  JobStatus = "archived"
)

// JobType represents the type of employment.
type JobType string

const (
	JobTypeFullTime   JobType = "full-time"
	JobTypePartTime   JobType = "part-time"
	JobTypeContract   JobType = "contract"
	JobTypeFreelance  JobType = "freelance"
	JobTypeInternship JobType = "internship"
)

// JobExperienceLevel represents the required experience level.
type JobExperienceLevel string

const (
	JobExperienceEntry  JobExperienceLevel = "entry"
	JobExperienceJunior JobExperienceLevel = "junior"
	JobExperienceMid    JobExperienceLevel = "mid"
	JobExperienceSenior JobExperienceLevel = "senior"
	JobExperienceLead   JobExperienceLevel = "lead"
)

type Job struct {
	ID               uuid.UUID          `json:"id" db:"id"`
	Title            string             `json:"title" db:"title"`
	Company          string             `json:"company" db:"company"`
	CompanyLogo      *string            `json:"company_logo,omitempty" db:"company_logo"`
	CompanyWebsite   *string            `json:"company_website,omitempty" db:"company_website"`
	CompanyLocation  *string            `json:"company_location,omitempty" db:"company_location"`
	Description      string             `json:"description" db:"description"`
	Requirements     string             `json:"requirements" db:"requirements"`
	Responsibilities *string            `json:"responsibilities,omitempty" db:"responsibilities"`
	Benefits         *string            `json:"benefits,omitempty" db:"benefits"`

	JobType          JobType            `json:"job_type" db:"job_type"`
	ExperienceLevel  JobExperienceLevel `json:"experience_level" db:"experience_level"`
	Location         *string            `json:"location,omitempty" db:"location"`
	RemotePossible   bool               `json:"remote_possible" db:"remote_possible"`
	SalaryMin        *int               `json:"salary_min,omitempty" db:"salary_min"`
	SalaryMax        *int               `json:"salary_max,omitempty" db:"salary_max"`
	SalaryCurrency   string             `json:"salary_currency" db:"salary_currency"`

	Status       JobStatus    `json:"status" db:"status"`
	PublishedAt  *time.Time   `json:"published_at,omitempty" db:"published_at"`
	ClosedAt     *time.Time   `json:"closed_at,omitempty" db:"closed_at"`
	ArchivedAt   *time.Time   `json:"archived_at,omitempty" db:"archived_at"`
	ExpiresAt    *time.Time   `json:"expires_at,omitempty" db:"expires_at"`

	PostedBy uuid.UUID `json:"posted_by" db:"posted_by"`

	ViewsCount        int `json:"views_count" db:"views_count"`
	ApplicationsCount int `json:"applications_count" db:"applications_count"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
