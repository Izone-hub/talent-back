package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/models"
)

// QuestionService handles question-related business logic.
type QuestionService struct {
	queries *database.Queries
}

// NewQuestionService creates a new QuestionService.
func NewQuestionService(db database.DBTX) *QuestionService {
	return &QuestionService{
		queries: database.New(db),
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

// CreateQuestion creates a new question and optionally its coding details.
func (s *QuestionService) CreateQuestion(ctx context.Context, userID uuid.UUID, req models.CreateQuestionRequest) (*models.QuestionResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// 1. Create the base question
	dbQuestion, err := s.queries.CreateQuestion(ctx, database.CreateQuestionParams{
		QuestionText:     req.QuestionText,
		QuestionType:     database.QuestionType(req.QuestionType),
		Difficulty:       database.QuestionDifficulty(req.Difficulty),
		Options:          req.Options,
		CorrectAnswer:    strPtrToPgText(req.CorrectAnswer),
		Explanation:      strPtrToPgText(req.Explanation),
		TimeLimitSeconds: intToPgInt4(req.TimeLimitSeconds),
		Points:           intToPgInt4(req.Points),
		Tags:             req.Tags,
		CreatedBy:        userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create question: %w", err)
	}

	res := &models.QuestionResponse{
		Question: dbQuestionToModel(dbQuestion),
	}

	// 2. If it's a coding challenge, create coding details
	if req.QuestionType == models.QuestionTypeCodingChallenge && req.CodingDetails != nil {
		dbCoding, err := s.queries.CreateCodingQuestion(ctx, database.CreateCodingQuestionParams{
			QuestionID:         dbQuestion.ID,
			Language:           req.CodingDetails.Language,
			CodeTemplate:       strPtrToPgText(req.CodingDetails.CodeTemplate),
			TestCases:          req.CodingDetails.TestCases,
			ExecutionTimeLimit: intToPgInt4(req.CodingDetails.ExecutionTimeLimit),
			MemoryLimit:        intToPgInt4(req.CodingDetails.MemoryLimit),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create coding details: %w", err)
		}
		codingModel := dbCodingToModel(dbCoding)
		res.CodingDetails = &codingModel
	}

	return res, nil
}

// ---------------------------------------------------------------------------
// Read
// ---------------------------------------------------------------------------

// GetQuestion fetches a question by ID along with its details.
func (s *QuestionService) GetQuestion(ctx context.Context, id uuid.UUID) (*models.QuestionResponse, error) {
	pgID := uuidToPgUUID(id)

	dbQuestion, err := s.queries.GetQuestionByID(ctx, pgID)
	if err != nil {
		return nil, fmt.Errorf("question not found: %w", err)
	}

	res := &models.QuestionResponse{
		Question: dbQuestionToModel(dbQuestion),
	}

	if res.QuestionType == models.QuestionTypeCodingChallenge {
		dbCoding, err := s.queries.GetCodingQuestion(ctx, pgID)
		if err == nil {
			codingModel := dbCodingToModel(dbCoding)
			res.CodingDetails = &codingModel
		}
	}

	return res, nil
}

// ListQuestions returns a paginated list of active questions.
func (s *QuestionService) ListQuestions(ctx context.Context, limit, offset int) ([]models.Question, error) {
	dbQuestions, err := s.queries.ListQuestions(ctx, database.ListQuestionsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list questions: %w", err)
	}

	questions := make([]models.Question, 0, len(dbQuestions))
	for _, q := range dbQuestions {
		questions = append(questions, dbQuestionToModel(q))
	}
	return questions, nil
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

// UpdateQuestion patches a question and its coding details.
func (s *QuestionService) UpdateQuestion(ctx context.Context, id uuid.UUID, req models.UpdateQuestionRequest) (*models.QuestionResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	pgID := uuidToPgUUID(id)

	// 1. Update base question
	params := database.UpdateQuestionParams{
		ID: pgID,
	}
	if req.QuestionText != nil {
		params.QuestionText = *req.QuestionText
	}
	params.Options = req.Options
	params.CorrectAnswer = strPtrToPgText(req.CorrectAnswer)
	params.Explanation = strPtrToPgText(req.Explanation)
	if req.TimeLimitSeconds != nil {
		params.TimeLimitSeconds = intToPgInt4(*req.TimeLimitSeconds)
	}
	if req.Points != nil {
		params.Points = intToPgInt4(*req.Points)
	}
	params.Tags = req.Tags

	dbQuestion, err := s.queries.UpdateQuestion(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update question: %w", err)
	}

	res := &models.QuestionResponse{
		Question: dbQuestionToModel(dbQuestion),
	}

	// 2. Update coding details if applicable
	if res.QuestionType == models.QuestionTypeCodingChallenge && req.CodingDetails != nil {
		codingParams := database.UpdateCodingQuestionParams{
			QuestionID: pgID,
		}
		if req.CodingDetails.Language != nil {
			codingParams.Language = *req.CodingDetails.Language
		}
		codingParams.CodeTemplate = strPtrToPgText(req.CodingDetails.CodeTemplate)
		codingParams.TestCases = req.CodingDetails.TestCases
		if req.CodingDetails.ExecutionTimeLimit != nil {
			codingParams.ExecutionTimeLimit = intToPgInt4(*req.CodingDetails.ExecutionTimeLimit)
		}
		if req.CodingDetails.MemoryLimit != nil {
			codingParams.MemoryLimit = intToPgInt4(*req.CodingDetails.MemoryLimit)
		}

		dbCoding, err := s.queries.UpdateCodingQuestion(ctx, codingParams)
		if err != nil {
			return nil, fmt.Errorf("failed to update coding details: %w", err)
		}
		codingModel := dbCodingToModel(dbCoding)
		res.CodingDetails = &codingModel
	}

	return res, nil
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

// DeleteQuestion soft-deletes a question.
func (s *QuestionService) DeleteQuestion(ctx context.Context, id uuid.UUID) error {
	return s.queries.DeleteQuestion(ctx, uuidToPgUUID(id))
}

// ---------------------------------------------------------------------------
// Helpers & Converters
// ---------------------------------------------------------------------------

func dbQuestionToModel(q database.Question) models.Question {
	var id uuid.UUID
	if q.ID.Valid {
		id, _ = uuid.FromBytes(q.ID.Bytes[:])
	}
	var createdBy *uuid.UUID
	if q.CreatedBy != uuid.Nil {
		createdBy = &q.CreatedBy
	}

	return models.Question{
		ID:               id,
		QuestionText:     q.QuestionText,
		QuestionType:     models.QuestionType(q.QuestionType),
		Difficulty:       models.QuestionDifficulty(q.Difficulty),
		Options:          json.RawMessage(q.Options),
		CorrectAnswer:    pgTextToStrPtr(q.CorrectAnswer),
		Explanation:      pgTextToStrPtr(q.Explanation),
		TimeLimitSeconds: int(q.TimeLimitSeconds.Int32),
		Points:           int(q.Points.Int32),
		Tags:             q.Tags,
		CreatedBy:        createdBy,
		IsActive:         q.IsActive.Bool,
		UsageCount:       int(q.UsageCount.Int32),
		CreatedAt:        pgTimestampToTime(q.CreatedAt),
		UpdatedAt:        pgTimestampToTime(q.UpdatedAt),
	}
}

func dbCodingToModel(c database.CodingQuestion) models.CodingQuestion {
	var id uuid.UUID
	if c.ID.Valid {
		id, _ = uuid.FromBytes(c.ID.Bytes[:])
	}
	var qID uuid.UUID
	if c.QuestionID.Valid {
		qID, _ = uuid.FromBytes(c.QuestionID.Bytes[:])
	}

	return models.CodingQuestion{
		ID:                 id,
		QuestionID:         qID,
		Language:           c.Language,
		CodeTemplate:       pgTextToStrPtr(c.CodeTemplate),
		TestCases:          json.RawMessage(c.TestCases),
		ExecutionTimeLimit: int(c.ExecutionTimeLimit.Int32),
		MemoryLimit:        int(c.MemoryLimit.Int32),
		CreatedAt:          pgTimestampToTime(c.CreatedAt),
	}
}
