package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/models"
)

type TagService struct {
	db *database.Queries
}

func NewTagService(db database.DBTX) *TagService {
	return &TagService{db: database.New(db)}
}

// CreateTag creates a new tag or updates an existing one if the name conflicts.
// Names are stored in lowercase for consistent matching.
func (s *TagService) CreateTag(ctx context.Context, req models.Tag) (database.Tag, error) {
	arg := database.CreateTagParams{
		Name:        strings.ToLower(req.Name),
		Category:    strPtrToNullCategory(req.Category),
		Description: strPtrToPgText(req.Description),
		Color:       strPtrToPgText(req.Color),
	}

	return s.db.CreateTag(ctx, arg)
}

// ListTags gets a paginated list of tags.
func (s *TagService) ListTags(ctx context.Context, limit, offset int32) ([]database.Tag, error) {
	return s.db.ListTags(ctx, database.ListTagsParams{
		Limit:  limit,
		Offset: offset,
	})
}

// GetTagByID fetches a single tag by UUID.
func (s *TagService) GetTagByID(ctx context.Context, id uuid.UUID) (database.Tag, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	return s.db.GetTagByID(ctx, pgID)
}

// AssignTagToJob assigns a specific tag to a specific job.
// Accepts either tagID (UUID) or tagName (string) — if tagID is zero, looks up by name.
func (s *TagService) AssignTagToJob(ctx context.Context, jobID uuid.UUID, tagID uuid.UUID, tagName string) (database.Tag, error) {
	if tagID == uuid.Nil && tagName != "" {
		tag, err := s.db.GetTagByName(ctx, strings.ToLower(tagName))
		if err != nil {
			return database.Tag{}, fmt.Errorf("tag not found by name %q: %w", tagName, err)
		}
		tagID = uuid.UUID(tag.ID.Bytes)
	}

	if tagID == uuid.Nil {
		return database.Tag{}, fmt.Errorf("tag_id or tag_name is required")
	}

	arg := database.AssignTagToJobParams{
		JobID: pgtype.UUID{Bytes: jobID, Valid: true},
		TagID: pgtype.UUID{Bytes: tagID, Valid: true},
	}
	if err := s.db.AssignTagToJob(ctx, arg); err != nil {
		return database.Tag{}, err
	}

	pgID := pgtype.UUID{Bytes: tagID, Valid: true}
	return s.db.GetTagByID(ctx, pgID)
}

// RemoveTagFromJob removes a specific tag from a specific job.
// Accepts either tagID (UUID) or tagName (string) — if tagID is zero, looks up by name.
func (s *TagService) RemoveTagFromJob(ctx context.Context, jobID uuid.UUID, tagID uuid.UUID, tagName string) error {
	if tagID == uuid.Nil && tagName != "" {
		tag, err := s.db.GetTagByName(ctx, strings.ToLower(tagName))
		if err != nil {
			return fmt.Errorf("tag not found by name %q: %w", tagName, err)
		}
		tagID = uuid.UUID(tag.ID.Bytes)
	}

	if tagID == uuid.Nil {
		return fmt.Errorf("tag_id or tag_name is required")
	}

	arg := database.RemoveTagFromJobParams{
		JobID: pgtype.UUID{Bytes: jobID, Valid: true},
		TagID: pgtype.UUID{Bytes: tagID, Valid: true},
	}
	return s.db.RemoveTagFromJob(ctx, arg)
}

// GetJobTags retrieves all tags associated with a specific job.
func (s *TagService) GetJobTags(ctx context.Context, jobID uuid.UUID) ([]database.Tag, error) {
	pgID := pgtype.UUID{Bytes: jobID, Valid: true}
	return s.db.GetJobTags(ctx, pgID)
}

// DeleteTag removes a tag by its ID.
func (s *TagService) DeleteTag(ctx context.Context, id uuid.UUID) error {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	return s.db.DeleteTag(ctx, pgID)
}

// UpdateTag modifies an existing tag.
func (s *TagService) UpdateTag(ctx context.Context, id uuid.UUID, req models.Tag) (database.Tag, error) {
	arg := database.UpdateTagParams{
		ID:          pgtype.UUID{Bytes: id, Valid: true},
		Name:        req.Name,
		Category:    strPtrToNullCategory(req.Category),
		Description: strPtrToPgText(req.Description),
		Color:       strPtrToPgText(req.Color),
	}

	return s.db.UpdateTag(ctx, arg)
}

// GetTagJobs retrieves all published jobs associated with a specific tag, paginated.
func (s *TagService) GetTagJobs(ctx context.Context, tagID uuid.UUID, limit, offset int32) ([]models.Job, error) {
	pgID := pgtype.UUID{Bytes: tagID, Valid: true}
	dbJobs, err := s.db.GetJobsByTag(ctx, database.GetJobsByTagParams{
		TagID:  pgID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	return dbJobsToModels(dbJobs), nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func strPtrToNullCategory(s *string) database.NullTagCategory {
	if s == nil {
		return database.NullTagCategory{Valid: false}
	}
	return database.NullTagCategory{
		TagCategory: database.TagCategory(*s),
		Valid:       true,
	}
}
