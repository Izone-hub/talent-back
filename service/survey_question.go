package service

import (
	"context"
	"fmt"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/models"
	"github.com/google/uuid"
)

// SurveyQuestionService handles CRUD for job screening questions.
type SurveyQuestionService struct {
	queries *database.Queries
}

// NewSurveyQuestionService creates a new SurveyQuestionService.
func NewSurveyQuestionService(db database.DBTX) *SurveyQuestionService {
	return &SurveyQuestionService{
		queries: database.New(db),
	}
}

// UpsertQuestions replaces all screening questions for a job in a single
// transaction. Pass an empty slice to clear all questions.
func (s *SurveyQuestionService) UpsertQuestions(ctx context.Context, jobID uuid.UUID, questions []models.SurveyQuestionRequest) ([]models.SurveyQuestion, error) {
	pgJobID := uuidToPgUUID(jobID)

	// Delete existing questions
	if err := s.queries.DeleteSurveyQuestionsByJob(ctx, pgJobID); err != nil {
		return nil, fmt.Errorf("failed to delete existing survey questions: %w", err)
	}

	// Insert new questions
	var result []models.SurveyQuestion
	for i, q := range questions {
		if q.QuestionText == "" {
			continue
		}
		dbQ, err := s.queries.CreateSurveyQuestion(ctx, database.CreateSurveyQuestionParams{
			JobID:          pgJobID,
			QuestionText:   q.QuestionText,
			ExpectedAnswer: q.ExpectedAnswer,
			SortOrder:      int32(i),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create survey question: %w", err)
		}
		result = append(result, dbSurveyQuestionToModel(dbQ))
	}

	if result == nil {
		result = []models.SurveyQuestion{}
	}
	return result, nil
}

// GetQuestions returns all screening questions for a job, ordered by sort_order.
func (s *SurveyQuestionService) GetQuestions(ctx context.Context, jobID uuid.UUID) ([]models.SurveyQuestion, error) {
	dbQs, err := s.queries.GetSurveyQuestionsByJob(ctx, uuidToPgUUID(jobID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch survey questions: %w", err)
	}

	result := make([]models.SurveyQuestion, 0, len(dbQs))
	for _, q := range dbQs {
		result = append(result, dbSurveyQuestionToModel(q))
	}
	return result, nil
}

// DeleteQuestion removes a single screening question.
func (s *SurveyQuestionService) DeleteQuestion(ctx context.Context, jobID, questionID uuid.UUID) error {
	return s.queries.DeleteSurveyQuestion(ctx, database.DeleteSurveyQuestionParams{
		ID:    uuidToPgUUID(questionID),
		JobID: uuidToPgUUID(jobID),
	})
}

func dbSurveyQuestionToModel(q database.JobSurveyQuestion) models.SurveyQuestion {
	var id, jobID uuid.UUID
	if q.ID.Valid {
		id, _ = uuid.FromBytes(q.ID.Bytes[:])
	}
	if q.JobID.Valid {
		jobID, _ = uuid.FromBytes(q.JobID.Bytes[:])
	}
	return models.SurveyQuestion{
		ID:             id,
		JobID:          jobID,
		QuestionText:   q.QuestionText,
		ExpectedAnswer: q.ExpectedAnswer,
		SortOrder:      int(q.SortOrder),
		CreatedAt:      pgTimestampToTime(q.CreatedAt),
		UpdatedAt:      pgTimestampToTime(q.UpdatedAt),
	}
}

