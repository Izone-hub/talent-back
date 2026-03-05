package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// JobService handles all job-related business logic.
type JobService struct {
	queries *database.Queries
}

// NewJobService creates a new JobService backed by the given database connection.
func NewJobService(db *pgx.Conn) *JobService {
	return &JobService{
		queries: database.New(db),
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

// CreateJob creates a new job in draft status. The caller's userID is set as
// the poster. Returns the created job.
func (s *JobService) CreateJob(ctx context.Context, userID uuid.UUID, req models.CreateJobRequest) (*models.Job, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	dbJob, err := s.queries.CreateJob(ctx, database.CreateJobParams{
		Title:            req.Title,
		Company:          req.Company,
		CompanyLogo:      strPtrToPgText(req.CompanyLogo),
		CompanyWebsite:   strPtrToPgText(req.CompanyWebsite),
		CompanyLocation:  strPtrToPgText(req.CompanyLocation),
		Description:      req.Description,
		Requirements:     req.Requirements,
		Responsibilities: strPtrToPgText(req.Responsibilities),
		Benefits:         strPtrToPgText(req.Benefits),
		JobType:          database.JobType(req.JobType),
		ExperienceLevel:  database.JobExperienceLevel(req.ExperienceLevel),
		Location:         strPtrToPgText(req.Location),
		RemotePossible:   pgtype.Bool{Bool: req.RemotePossible, Valid: true},
		SalaryMin:        intPtrToPgInt4(req.SalaryMin),
		SalaryMax:        intPtrToPgInt4(req.SalaryMax),
		SalaryCurrency:   strToPgText(req.SalaryCurrency),
		Status:           database.JobStatusDraft,
		PostedBy:         uuidToPgUUID(userID),
		ExpiresAt:        timePtrToPgTimestamp(req.ExpiresAt),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	job := dbJobToModel(dbJob)
	return &job, nil
}

// ---------------------------------------------------------------------------
// Read
// ---------------------------------------------------------------------------

// GetJob returns any job by ID regardless of status (for owner/admin usage).
func (s *JobService) GetJob(ctx context.Context, jobID uuid.UUID) (*models.Job, error) {
	dbJob, err := s.queries.GetJobByID(ctx, uuidToPgUUID(jobID))
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}
	job := dbJobToModel(dbJob)
	return &job, nil
}

// GetPublishedJob returns a published job and increments its view count.
// This is the endpoint used by applicants browsing jobs.
func (s *JobService) GetPublishedJob(ctx context.Context, jobID uuid.UUID) (*models.Job, error) {
	pgID := uuidToPgUUID(jobID)

	dbJob, err := s.queries.GetPublishedJobByID(ctx, pgID)
	if err != nil {
		return nil, fmt.Errorf("published job not found: %w", err)
	}

	// Fire-and-forget view increment (best-effort, non-blocking).
	_ = s.queries.IncrementJobViews(ctx, pgID)

	job := dbJobToModel(dbJob)
	return &job, nil
}

// ListPublishedJobs returns a paginated list of published jobs for the public
// job board.
func (s *JobService) ListPublishedJobs(ctx context.Context, limit, offset int) ([]models.Job, error) {
	dbJobs, err := s.queries.ListPublishedJobs(ctx, database.ListPublishedJobsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list published jobs: %w", err)
	}
	return dbJobsToModels(dbJobs), nil
}

// ListMyJobs returns jobs posted by a specific user (any status), paginated.
func (s *JobService) ListMyJobs(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Job, error) {
	dbJobs, err := s.queries.ListJobsByPoster(ctx, database.ListJobsByPosterParams{
		PostedBy: uuidToPgUUID(userID),
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	return dbJobsToModels(dbJobs), nil
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

// UpdateJob patches a job's metadata. Only the job owner can update, and only
// while the job is still in draft status.
func (s *JobService) UpdateJob(ctx context.Context, userID, jobID uuid.UUID, req models.UpdateJobRequest) (*models.Job, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Build params — COALESCE in SQL makes empty-string → keep-old, so we
	// send the current value or the new one depending on what's provided.
	params := database.UpdateJobParams{
		ID:       uuidToPgUUID(jobID),
		PostedBy: uuidToPgUUID(userID),
	}

	// For required string fields, pass empty string to let COALESCE keep old value
	if req.Title != nil {
		params.Title = *req.Title
	}
	if req.Company != nil {
		params.Company = *req.Company
	}
	if req.Description != nil {
		params.Description = *req.Description
	}
	if req.Requirements != nil {
		params.Requirements = *req.Requirements
	}
	if req.JobType != nil {
		params.JobType = database.JobType(*req.JobType)
	}
	if req.ExperienceLevel != nil {
		params.ExperienceLevel = database.JobExperienceLevel(*req.ExperienceLevel)
	}
	if req.Location != nil {
		params.Location = strToPgText(*req.Location)
	}
	if req.RemotePossible != nil {
		params.RemotePossible = pgtype.Bool{Bool: *req.RemotePossible, Valid: true}
	}
	if req.SalaryMin != nil {
		params.SalaryMin = pgtype.Int4{Int32: int32(*req.SalaryMin), Valid: true}
	}
	if req.SalaryMax != nil {
		params.SalaryMax = pgtype.Int4{Int32: int32(*req.SalaryMax), Valid: true}
	}

	dbJob, err := s.queries.UpdateJob(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update job: %w", err)
	}

	job := dbJobToModel(dbJob)
	return &job, nil
}

// ---------------------------------------------------------------------------
// Lifecycle transitions
// ---------------------------------------------------------------------------

// PublishJob transitions a job from Draft → Published. Only the owner can do
// this. The SQL query enforces the status prerequisite.
func (s *JobService) PublishJob(ctx context.Context, userID, jobID uuid.UUID) (*models.Job, error) {
	dbJob, err := s.queries.PublishJob(ctx, database.PublishJobParams{
		ID:       uuidToPgUUID(jobID),
		PostedBy: uuidToPgUUID(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to publish job (must be in draft status and owned by you): %w", err)
	}
	job := dbJobToModel(dbJob)
	return &job, nil
}

// CloseJob transitions a job from Published → Closed. Only the owner can do
// this. New applications will be rejected once a job is closed.
func (s *JobService) CloseJob(ctx context.Context, userID, jobID uuid.UUID) (*models.Job, error) {
	dbJob, err := s.queries.CloseJob(ctx, database.CloseJobParams{
		ID:       uuidToPgUUID(jobID),
		PostedBy: uuidToPgUUID(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to close job (must be published and owned by you): %w", err)
	}
	job := dbJobToModel(dbJob)
	return &job, nil
}

// ArchiveJob transitions a job from Published/Closed → Archived. The job is
// effectively hidden from all listings and no longer accepts applications.
func (s *JobService) ArchiveJob(ctx context.Context, userID, jobID uuid.UUID) (*models.Job, error) {
	dbJob, err := s.queries.ArchiveJob(ctx, database.ArchiveJobParams{
		ID:       uuidToPgUUID(jobID),
		PostedBy: uuidToPgUUID(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to archive job (must be published or closed and owned by you): %w", err)
	}
	job := dbJobToModel(dbJob)
	return &job, nil
}

// ---------------------------------------------------------------------------
// Conversion helpers: database.Job (pgtype) ↔ models.Job (Go stdlib types)
// ---------------------------------------------------------------------------

func dbJobToModel(j database.Job) models.Job {
	var id uuid.UUID
	if j.ID.Valid {
		id, _ = uuid.FromBytes(j.ID.Bytes[:])
	}
	var postedBy uuid.UUID
	if j.PostedBy.Valid {
		postedBy, _ = uuid.FromBytes(j.PostedBy.Bytes[:])
	}

	return models.Job{
		ID:                id,
		Title:             j.Title,
		Company:           j.Company,
		CompanyLogo:       pgTextToStrPtr(j.CompanyLogo),
		CompanyWebsite:    pgTextToStrPtr(j.CompanyWebsite),
		CompanyLocation:   pgTextToStrPtr(j.CompanyLocation),
		Description:       j.Description,
		Requirements:      j.Requirements,
		Responsibilities:  pgTextToStrPtr(j.Responsibilities),
		Benefits:          pgTextToStrPtr(j.Benefits),
		JobType:           models.JobType(j.JobType),
		ExperienceLevel:   models.JobExperienceLevel(j.ExperienceLevel),
		Location:          pgTextToStrPtr(j.Location),
		RemotePossible:    j.RemotePossible.Bool,
		SalaryMin:         pgInt4ToIntPtr(j.SalaryMin),
		SalaryMax:         pgInt4ToIntPtr(j.SalaryMax),
		SalaryCurrency:    pgTextToString(j.SalaryCurrency),
		Status:            models.JobStatus(j.Status),
		PublishedAt:       pgTimestampToTimePtr(j.PublishedAt),
		ClosedAt:          pgTimestampToTimePtr(j.ClosedAt),
		ArchivedAt:        pgTimestampToTimePtr(j.ArchivedAt),
		ExpiresAt:         pgTimestampToTimePtr(j.ExpiresAt),
		PostedBy:          postedBy,
		ViewsCount:        int(j.ViewsCount.Int32),
		ApplicationsCount: int(j.ApplicationsCount.Int32),
		CreatedAt:         pgTimestampToTime(j.CreatedAt),
		UpdatedAt:         pgTimestampToTime(j.UpdatedAt),
	}
}

func dbJobsToModels(dbJobs []database.Job) []models.Job {
	jobs := make([]models.Job, 0, len(dbJobs))
	for _, j := range dbJobs {
		jobs = append(jobs, dbJobToModel(j))
	}
	return jobs
}

// ---------------------------------------------------------------------------
// pgtype ↔ Go type helpers (job-specific; generic ones live in auth.go)
// ---------------------------------------------------------------------------

func strPtrToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func intPtrToPgInt4(i *int) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: int32(*i), Valid: true}
}

func pgInt4ToIntPtr(i pgtype.Int4) *int {
	if !i.Valid {
		return nil
	}
	v := int(i.Int32)
	return &v
}

func uuidToPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func timePtrToPgTimestamp(t *time.Time) pgtype.Timestamp {
	if t == nil {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: *t, Valid: true}
}
