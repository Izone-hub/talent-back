package models

import (
	"time"

	"github.com/google/uuid"
)

// ApplicationStatus represents the status of a job application.
type ApplicationStatus string

const (
	ApplicationStatusDraft         ApplicationStatus = "draft"
	ApplicationStatusSubmitted     ApplicationStatus = "submitted"
	ApplicationStatusQuizStarted   ApplicationStatus = "quiz_started"
	ApplicationStatusQuizCompleted ApplicationStatus = "quiz_completed"
	ApplicationStatusUnderReview   ApplicationStatus = "under_review"
	ApplicationStatusShortlisted   ApplicationStatus = "shortlisted"
	ApplicationStatusInterviewed   ApplicationStatus = "interviewed"
	ApplicationStatusRejected      ApplicationStatus = "rejected"
	ApplicationStatusAccepted      ApplicationStatus = "accepted"
	ApplicationStatusWithdrawn     ApplicationStatus = "withdrawn"
)

type JobApplication struct {
	ID     uuid.UUID `json:"id" db:"id"`
	JobID  uuid.UUID `json:"job_id" db:"job_id"`
	UserID uuid.UUID `json:"user_id" db:"user_id"`

	GithubUsername     string  `json:"github_username" db:"github_username"`
	GithubID           int64   `json:"github_id" db:"github_id"`
	ApplicantEmail     *string `json:"applicant_email,omitempty" db:"applicant_email"`
	ApplicantName      *string `json:"applicant_name,omitempty" db:"applicant_name"`
	ApplicantAvatarURL *string `json:"applicant_avatar_url,omitempty" db:"applicant_avatar_url"`

	CoverLetter            *string    `json:"cover_letter,omitempty" db:"cover_letter"`
	ProposedSalary         *int       `json:"proposed_salary,omitempty" db:"proposed_salary"`
	ProposedSalaryCurrency string     `json:"proposed_salary_currency" db:"proposed_salary_currency"`
	AvailabilityDate       *time.Time `json:"availability_date,omitempty" db:"availability_date"`
	PortfolioURL           *string    `json:"portfolio_url,omitempty" db:"portfolio_url"`
	LinkedinURL            *string    `json:"linkedin_url,omitempty" db:"linkedin_url"`
	Notes                  *string    `json:"notes,omitempty" db:"notes"`

	Status      ApplicationStatus `json:"status" db:"status"`
	SubmittedAt *time.Time        `json:"submitted_at,omitempty" db:"submitted_at"`

	ReviewedAt       *time.Time `json:"reviewed_at,omitempty" db:"reviewed_at"`
	ReviewedBy       *uuid.UUID `json:"reviewed_by,omitempty" db:"reviewed_by"`
	EmployerFeedback *string    `json:"employer_feedback,omitempty" db:"employer_feedback"`
	RejectionReason  *string    `json:"rejection_reason,omitempty" db:"rejection_reason"`

	QuizID          *uuid.UUID `json:"quiz_id,omitempty" db:"quiz_id"`
	QuizScore       *int       `json:"quiz_score,omitempty" db:"quiz_score"`
	QuizCompletedAt *time.Time `json:"quiz_completed_at,omitempty" db:"quiz_completed_at"`
	QuizPassed      *bool      `json:"quiz_passed,omitempty" db:"quiz_passed"`

	CanViewAISummary bool `json:"can_view_ai_summary" db:"can_view_ai_summary"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
