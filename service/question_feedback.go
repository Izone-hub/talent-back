package service

import (
	"context"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type QuestionFeedbackService struct{ queries *database.Queries }

func NewQuestionFeedbackService(db database.DBTX) *QuestionFeedbackService {
	return &QuestionFeedbackService{queries: database.New(db)}
}

func feedbackUUID(id uuid.UUID) pgtype.UUID {
	var value pgtype.UUID
	copy(value.Bytes[:], id[:])
	value.Valid = true
	return value
}

func (s *QuestionFeedbackService) Get(ctx context.Context, userID, questionID uuid.UUID) (string, error) {
	return s.queries.GetQuestionFeedback(ctx, database.GetQuestionFeedbackParams{UserID: feedbackUUID(userID), QuestionID: feedbackUUID(questionID)})
}

func (s *QuestionFeedbackService) Upsert(ctx context.Context, userID, questionID uuid.UUID, feedback string) (string, error) {
	return s.queries.UpsertQuestionFeedback(ctx, database.UpsertQuestionFeedbackParams{UserID: feedbackUUID(userID), QuestionID: feedbackUUID(questionID), Feedback: feedback})
}
