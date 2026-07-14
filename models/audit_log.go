package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditAction represents the type of audited action.
type AuditAction string

const (
	AuditActionUserLogin                AuditAction = "user_login"
	AuditActionUserLogout               AuditAction = "user_logout"
	AuditActionUserRegistered           AuditAction = "user_registered"
	AuditActionUserUpdated              AuditAction = "user_updated"
	AuditActionUserDeleted              AuditAction = "user_deleted"
	AuditActionRoleChanged              AuditAction = "role_changed"
	AuditActionJobCreated               AuditAction = "job_created"
	AuditActionJobUpdated               AuditAction = "job_updated"
	AuditActionJobPublished             AuditAction = "job_published"
	AuditActionJobClosed                AuditAction = "job_closed"
	AuditActionJobArchived              AuditAction = "job_archived"
	AuditActionJobDeleted               AuditAction = "job_deleted"
	AuditActionApplicationSubmitted     AuditAction = "application_submitted"
	AuditActionApplicationUpdated       AuditAction = "application_updated"
	AuditActionApplicationStatusChanged AuditAction = "application_status_changed"
	AuditActionApplicationWithdrawn     AuditAction = "application_withdrawn"
	AuditActionQuizStarted              AuditAction = "quiz_started"
	AuditActionQuizCompleted            AuditAction = "quiz_completed"
	AuditActionQuizAutoSaved            AuditAction = "quiz_auto_saved"
	AuditActionQuizTimedOut             AuditAction = "quiz_timed_out"
	AuditActionCvUploaded               AuditAction = "cv_uploaded"
	AuditActionCvDeleted                AuditAction = "cv_deleted"
	AuditActionCvUpdated                AuditAction = "cv_updated"
	AuditActionAdminLogin               AuditAction = "admin_login"
	AuditActionAdminAction              AuditAction = "admin_action"
	AuditActionSettingsChanged          AuditAction = "settings_changed"
	AuditActionReportGenerated          AuditAction = "report_generated"
)

type AuditLog struct {
	ID                uuid.UUID       `json:"id" db:"id"`
	UserID            *uuid.UUID      `json:"user_id,omitempty" db:"user_id"`
	UserEmail         *string         `json:"user_email,omitempty" db:"user_email"`
	UserRole          *string         `json:"user_role,omitempty" db:"user_role"`
	Action            AuditAction     `json:"action" db:"action"`
	EntityType        *string         `json:"entity_type,omitempty" db:"entity_type"`
	EntityID          *uuid.UUID      `json:"entity_id,omitempty" db:"entity_id"`
	PerformedAt       time.Time       `json:"performed_at" db:"performed_at"`
	IPAddress         *string         `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent         *string         `json:"user_agent,omitempty" db:"user_agent"`
	Details           json.RawMessage `json:"details,omitempty" db:"details"`
	OldValues         json.RawMessage `json:"old_values,omitempty" db:"old_values"`
	NewValues         json.RawMessage `json:"new_values,omitempty" db:"new_values"`
	Status            string          `json:"status" db:"status"`
	ErrorMessage      *string         `json:"error_message,omitempty" db:"error_message"`
	DataRetentionDays *int            `json:"data_retention_days,omitempty" db:"data_retention_days"`
	MarkedForDeletion bool            `json:"marked_for_deletion" db:"marked_for_deletion"`
}

type AdminAuditTrail struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	AuditLogID       uuid.UUID  `json:"audit_log_id" db:"audit_log_id"`
	AdminID          uuid.UUID  `json:"admin_id" db:"admin_id"`
	AdminNotes       *string    `json:"admin_notes,omitempty" db:"admin_notes"`
	ApprovalRequired bool       `json:"approval_required" db:"approval_required"`
	ApprovedBy       *uuid.UUID `json:"approved_by,omitempty" db:"approved_by"`
	ApprovedAt       *time.Time `json:"approved_at,omitempty" db:"approved_at"`
	SensitivityLevel string     `json:"sensitivity_level" db:"sensitivity_level"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

type AuditExport struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	ExportedBy      *uuid.UUID      `json:"exported_by,omitempty" db:"exported_by"`
	ExportType      *string         `json:"export_type,omitempty" db:"export_type"`
	DateRangeStart  *time.Time      `json:"date_range_start,omitempty" db:"date_range_start"`
	DateRangeEnd    *time.Time      `json:"date_range_end,omitempty" db:"date_range_end"`
	FilterCriteria  json.RawMessage `json:"filter_criteria,omitempty" db:"filter_criteria"`
	FilePath        *string         `json:"file_path,omitempty" db:"file_path"`
	ExportedAt      time.Time       `json:"exported_at" db:"exported_at"`
	DownloadedCount int             `json:"downloaded_count" db:"downloaded_count"`
}
