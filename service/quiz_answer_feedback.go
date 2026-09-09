package service

import (
	"context"
	"time"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/google/uuid"
)

type QuizAnswerFeedbackService struct {
	queries *database.Queries
}

func NewQuizAnswerFeedbackService(db database.DBTX) *QuizAnswerFeedbackService {
	return &QuizAnswerFeedbackService{queries: database.New(db)}
}

type QuizAnswerFeedbackResponse struct {
	ID            string `json:"id"`
	UserID        string `json:"user_id"`
	QuizAttemptID string `json:"quiz_attempt_id"`
	QuestionID    string `json:"question_id"`
	ApplicationID string `json:"application_id"`
	Feedback      string `json:"feedback"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

func quizAnswerFeedbackResponse(row database.QuizAnswerFeedback) QuizAnswerFeedbackResponse {
	return QuizAnswerFeedbackResponse{
		ID:            uuid.UUID(row.ID.Bytes).String(),
		UserID:        row.UserID.String(),
		QuizAttemptID: row.QuizAttemptID.String(),
		QuestionID:    row.QuestionID.String(),
		ApplicationID: row.ApplicationID.String(),
		Feedback:      row.Feedback,
		CreatedAt:     row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:     row.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func (s *QuizAnswerFeedbackService) Get(ctx context.Context, userID, quizAttemptID, questionID uuid.UUID) (*QuizAnswerFeedbackResponse, error) {
	item, err := s.queries.GetQuizAnswerFeedback(ctx, database.GetQuizAnswerFeedbackParams{
		UserID:        feedbackUUID(userID),
		QuizAttemptID: feedbackUUID(quizAttemptID),
		QuestionID:    feedbackUUID(questionID),
	})
	if err != nil {
		return nil, err
	}
	resp := quizAnswerFeedbackResponse(item)
	return &resp, nil
}

func (s *QuizAnswerFeedbackService) GetByAttempt(ctx context.Context, userID, quizAttemptID uuid.UUID) ([]QuizAnswerFeedbackResponse, error) {
	items, err := s.queries.GetQuizAnswerFeedbackByAttempt(ctx, database.GetQuizAnswerFeedbackByAttemptParams{
		UserID:        feedbackUUID(userID),
		QuizAttemptID: feedbackUUID(quizAttemptID),
	})
	if err != nil {
		return nil, err
	}
	responses := make([]QuizAnswerFeedbackResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, quizAnswerFeedbackResponse(item))
	}
	return responses, nil
}

func (s *QuizAnswerFeedbackService) Upsert(ctx context.Context, userID, quizAttemptID, questionID, applicationID uuid.UUID, feedback string) (*QuizAnswerFeedbackResponse, error) {
	item, err := s.queries.UpsertQuizAnswerFeedback(ctx, database.UpsertQuizAnswerFeedbackParams{
		UserID:        feedbackUUID(userID),
		QuizAttemptID: feedbackUUID(quizAttemptID),
		QuestionID:    feedbackUUID(questionID),
		ApplicationID: feedbackUUID(applicationID),
		Feedback:      feedback,
	})
	if err != nil {
		return nil, err
	}
	resp := quizAnswerFeedbackResponse(item)
	return &resp, nil
}

func (s *QuizAnswerFeedbackService) Delete(ctx context.Context, userID, quizAttemptID, questionID uuid.UUID) error {
	return s.queries.DeleteQuizAnswerFeedback(ctx, database.DeleteQuizAnswerFeedbackParams{
		UserID:        feedbackUUID(userID),
		QuizAttemptID: feedbackUUID(quizAttemptID),
		QuestionID:    feedbackUUID(questionID),
	})
}

func (s *QuizAnswerFeedbackService) GetByApplication(ctx context.Context, applicationID uuid.UUID) ([]QuizAnswerFeedbackResponse, error) {
	items, err := s.queries.ListQuizAnswerFeedbackByApplication(ctx, feedbackUUID(applicationID))
	if err != nil {
		return nil, err
	}
	responses := make([]QuizAnswerFeedbackResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, quizAnswerFeedbackResponse(item))
	}
	return responses, nil
}
