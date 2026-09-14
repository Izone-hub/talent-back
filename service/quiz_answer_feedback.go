package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/google/uuid"
)

type QuizAnswerFeedbackService struct {
	queries *database.Queries
	db      database.DBTX
}

func NewQuizAnswerFeedbackService(db database.DBTX) *QuizAnswerFeedbackService {
	return &QuizAnswerFeedbackService{queries: database.New(db), db: db}
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
	// 1. Verify quiz attempt ownership and application match
	var (
		attemptUserID uuid.UUID
		attemptAppID  uuid.UUID
	)
	err := s.db.QueryRow(ctx, `
		SELECT user_id, application_id FROM quiz_attempts WHERE id = $1
	`, quizAttemptID).Scan(&attemptUserID, &attemptAppID)
	if err != nil {
		return nil, fmt.Errorf("quiz attempt not found: %w", err)
	}
	if attemptUserID != userID {
		return nil, fmt.Errorf("quiz attempt does not belong to this user")
	}
	if attemptAppID != applicationID {
		return nil, fmt.Errorf("application does not match quiz attempt")
	}

	// 2. Verify question belongs to attempt
	var dummy int
	err = s.db.QueryRow(ctx, `
		SELECT 1 FROM quiz_attempt_questions WHERE quiz_attempt_id = $1 AND question_id = $2
	`, quizAttemptID, questionID).Scan(&dummy)
	if err != nil {
		return nil, fmt.Errorf("question does not belong to this quiz attempt")
	}

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

func (s *QuizAnswerFeedbackService) GetByApplication(ctx context.Context, applicationID, requesterID uuid.UUID, isAdmin bool) ([]QuizAnswerFeedbackResponse, error) {
	if !isAdmin {
		var appUserID uuid.UUID
		err := s.db.QueryRow(ctx, `SELECT user_id FROM job_applications WHERE id = $1`, applicationID).Scan(&appUserID)
		if err != nil {
			return nil, fmt.Errorf("application not found: %w", err)
		}
		if appUserID != requesterID {
			return nil, fmt.Errorf("application does not belong to this user")
		}
	}

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
