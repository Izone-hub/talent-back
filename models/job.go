package models

import (
	"fmt"
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

// validJobTypes is the set of allowed job types.
var validJobTypes = map[JobType]bool{
	JobTypeFullTime: true, JobTypePartTime: true,
	JobTypeContract: true, JobTypeFreelance: true,
	JobTypeInternship: true,
}

// validExperienceLevels is the set of allowed experience levels.
var validExperienceLevels = map[JobExperienceLevel]bool{
	JobExperienceEntry: true, JobExperienceJunior: true,
	JobExperienceMid: true, JobExperienceSenior: true,
	JobExperienceLead: true,
}

// ---------------------------------------------------------------------------
// Core Model
// ---------------------------------------------------------------------------

type Job struct {
	ID               uuid.UUID `json:"id" db:"id"`
	Title            string    `json:"title" db:"title"`
	Company          string    `json:"company" db:"company"`
	CompanyLogo      *string   `json:"company_logo,omitempty" db:"company_logo"`
	CompanyWebsite   *string   `json:"company_website,omitempty" db:"company_website"`
	CompanyLocation  *string   `json:"company_location,omitempty" db:"company_location"`
	Description      string    `json:"description" db:"description"`
	Requirements     string    `json:"requirements" db:"requirements"`
	Responsibilities *string   `json:"responsibilities,omitempty" db:"responsibilities"`
	Benefits         *string   `json:"benefits,omitempty" db:"benefits"`

	JobType         JobType            `json:"job_type" db:"job_type"`
	ExperienceLevel JobExperienceLevel `json:"experience_level" db:"experience_level"`
	Location        *string            `json:"location,omitempty" db:"location"`
	RemotePossible  bool               `json:"remote_possible" db:"remote_possible"`
	SalaryMin       *int               `json:"salary_min,omitempty" db:"salary_min"`
	SalaryMax       *int               `json:"salary_max,omitempty" db:"salary_max"`
	SalaryCurrency  string             `json:"salary_currency" db:"salary_currency"`

	Status      JobStatus  `json:"status" db:"status"`
	PublishedAt *time.Time `json:"published_at,omitempty" db:"published_at"`
	ClosedAt    *time.Time `json:"closed_at,omitempty" db:"closed_at"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty" db:"archived_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`

	PostedBy uuid.UUID `json:"posted_by" db:"posted_by"`

	ViewsCount        int `json:"views_count" db:"views_count"`
	ApplicationsCount int `json:"applications_count" db:"applications_count"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Tags      []Tag     `json:"tags,omitempty" db:"-"`

	// UserApplication contains application info if the current user has applied
	UserApplication *JobUserApplication `json:"user_application,omitempty" db:"-"`
}

// JobUserApplication holds the user's application status for a job
type JobUserApplication struct {
	Applied       bool       `json:"applied"`
	ApplicationID *uuid.UUID `json:"application_id,omitempty"`
	Status        *string    `json:"status,omitempty"`
	SubmittedAt   *time.Time `json:"submitted_at,omitempty"`
}

// ---------------------------------------------------------------------------
// Request DTOs
// ---------------------------------------------------------------------------

// CreateJobRequest is the payload sent by an admin/poster to create a new job.
type CreateJobRequest struct {
	Title            string             `json:"title"`
	Company          string             `json:"company"`
	CompanyLogo      *string            `json:"company_logo,omitempty"`
	CompanyWebsite   *string            `json:"company_website,omitempty"`
	CompanyLocation  *string            `json:"company_location,omitempty"`
	Description      string             `json:"description"`
	Requirements     string             `json:"requirements"`
	Responsibilities *string            `json:"responsibilities,omitempty"`
	Benefits         *string            `json:"benefits,omitempty"`
	JobType          JobType            `json:"job_type"`
	ExperienceLevel  JobExperienceLevel `json:"experience_level"`
	Location         *string            `json:"location,omitempty"`
	RemotePossible   bool               `json:"remote_possible"`
	SalaryMin        *int               `json:"salary_min,omitempty"`
	SalaryMax        *int               `json:"salary_max,omitempty"`
	SalaryCurrency   string             `json:"salary_currency"`
	ExpiresAt        *time.Time         `json:"expires_at,omitempty"`
}

// Validate checks that all required fields are present and valid.
func (r *CreateJobRequest) Validate() error {
	if r.Title == "" {
		return fmt.Errorf("title is required")
	}
	if r.Company == "" {
		return fmt.Errorf("company is required")
	}
	if r.Description == "" {
		return fmt.Errorf("description is required")
	}
	if r.Requirements == "" {
		return fmt.Errorf("requirements is required")
	}
	if !validJobTypes[r.JobType] {
		return fmt.Errorf("invalid job_type: %s", r.JobType)
	}
	if !validExperienceLevels[r.ExperienceLevel] {
		return fmt.Errorf("invalid experience_level: %s", r.ExperienceLevel)
	}
	if r.SalaryMin != nil && r.SalaryMax != nil && *r.SalaryMin > *r.SalaryMax {
		return fmt.Errorf("salary_min cannot be greater than salary_max")
	}
	if r.SalaryCurrency == "" {
		r.SalaryCurrency = "USD"
	}
	return nil
}

// UpdateJobRequest is the payload for partially updating a draft job.
// All fields are optional — only non-nil fields will be applied.
type UpdateJobRequest struct {
	Title           *string             `json:"title,omitempty"`
	Company         *string             `json:"company,omitempty"`
	Description     *string             `json:"description,omitempty"`
	Requirements    *string             `json:"requirements,omitempty"`
	JobType         *JobType            `json:"job_type,omitempty"`
	ExperienceLevel *JobExperienceLevel `json:"experience_level,omitempty"`
	Location        *string             `json:"location,omitempty"`
	RemotePossible  *bool               `json:"remote_possible,omitempty"`
	SalaryMin       *int                `json:"salary_min,omitempty"`
	SalaryMax       *int                `json:"salary_max,omitempty"`
}

// Validate checks that any supplied enum values are valid.
func (r *UpdateJobRequest) Validate() error {
	if r.JobType != nil && !validJobTypes[*r.JobType] {
		return fmt.Errorf("invalid job_type: %s", *r.JobType)
	}
	if r.ExperienceLevel != nil && !validExperienceLevels[*r.ExperienceLevel] {
		return fmt.Errorf("invalid experience_level: %s", *r.ExperienceLevel)
	}
	if r.SalaryMin != nil && r.SalaryMax != nil && *r.SalaryMin > *r.SalaryMax {
		return fmt.Errorf("salary_min cannot be greater than salary_max")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

// JobResponse is the clean API response sent to clients.
type JobResponse struct {
	ID                uuid.UUID          `json:"id"`
	Title             string             `json:"title"`
	Company           string             `json:"company"`
	CompanyLogo       *string            `json:"company_logo,omitempty"`
	CompanyWebsite    *string            `json:"company_website,omitempty"`
	CompanyLocation   *string            `json:"company_location,omitempty"`
	Description       string             `json:"description"`
	Requirements      string             `json:"requirements"`
	Responsibilities  *string            `json:"responsibilities,omitempty"`
	Benefits          *string            `json:"benefits,omitempty"`
	JobType           JobType            `json:"job_type"`
	ExperienceLevel   JobExperienceLevel `json:"experience_level"`
	Location          *string            `json:"location,omitempty"`
	RemotePossible    bool               `json:"remote_possible"`
	SalaryMin         *int               `json:"salary_min,omitempty"`
	SalaryMax         *int               `json:"salary_max,omitempty"`
	SalaryCurrency    string             `json:"salary_currency"`
	Status            JobStatus          `json:"status"`
	PublishedAt       *time.Time         `json:"published_at,omitempty"`
	ClosedAt          *time.Time         `json:"closed_at,omitempty"`
	ExpiresAt         *time.Time         `json:"expires_at,omitempty"`
	ViewsCount        int                `json:"views_count"`
	ApplicationsCount int                `json:"applications_count"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
	Tags              []Tag              `json:"tags,omitempty"`
	UserApplication   *JobUserApplication `json:"user_application,omitempty"`
}

// JobListResponse wraps a paginated list of jobs.
type JobListResponse struct {
	Jobs   []JobResponse `json:"jobs"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

// ToResponse converts a Job model to a clean API response.
func (j *Job) ToResponse() JobResponse {
	return JobResponse{
		ID:                j.ID,
		Title:             j.Title,
		Company:           j.Company,
		CompanyLogo:       j.CompanyLogo,
		CompanyWebsite:    j.CompanyWebsite,
		CompanyLocation:   j.CompanyLocation,
		Description:       j.Description,
		Requirements:      j.Requirements,
		Responsibilities:  j.Responsibilities,
		Benefits:          j.Benefits,
		JobType:           j.JobType,
		ExperienceLevel:   j.ExperienceLevel,
		Location:          j.Location,
		RemotePossible:    j.RemotePossible,
		SalaryMin:         j.SalaryMin,
		SalaryMax:         j.SalaryMax,
		SalaryCurrency:    j.SalaryCurrency,
		Status:            j.Status,
		PublishedAt:       j.PublishedAt,
		ClosedAt:          j.ClosedAt,
		ExpiresAt:         j.ExpiresAt,
		ViewsCount:        j.ViewsCount,
		ApplicationsCount: j.ApplicationsCount,
		CreatedAt:         j.CreatedAt,
		UpdatedAt:         j.UpdatedAt,
		Tags:              j.Tags,
		UserApplication:   j.UserApplication,
	}
}
