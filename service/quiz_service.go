package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QuizAttempt represents a quiz attempt structure in the database
type QuizAttempt struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Title     string    `json:"title"`
	Type      string    `json:"type"` // Tags for Gemini microservice (comma-separated)
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// QuizService handles all quiz logic and database calls
type QuizService struct {
	pool *pgxpool.Pool
}

// NewQuizService receives the *pgxpool.Pool sent from main.go
func NewQuizService(pool *pgxpool.Pool) *QuizService {
	return &QuizService{pool: pool}
}

func (s *QuizService) GetUserQuizzes(ctx context.Context, userID string) ([]QuizAttempt, error) {
	return []QuizAttempt{}, nil
}

func (s *QuizService) GetQuizAttempt(ctx context.Context, attemptID string, userID string) (*QuizAttempt, error) {
	return &QuizAttempt{
		ID:     uuid.MustParse(attemptID),
		Title:  "Technical Core Assessment",
		Type:   "go,svelte,postgresql",
		Status: "active",
	}, nil
}

func (s *QuizService) StartQuizAttempt(ctx context.Context, attemptID string, userID string) error {
	return nil
}

// 🔥 Fix: Removed 'simulation' and used the correct Go interface format
func (s *QuizService) GetQuizQuestions(ctx context.Context, attemptID string, userID string, tags []string) (interface{}, error) {
	// Query to extract real questions from the database
	rows, err := s.pool.Query(ctx, `
		SELECT id, question_text, question_type, options 
		FROM questions 
		ORDER BY RANDOM() LIMIT 10`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []map[string]interface{}
	for rows.Next() {
		var id uuid.UUID
		var qText, qType string
		var options *string // Can be a JSON string

		err := rows.Scan(&id, &qText, &qType, &options)
		if err != nil {
			return nil, err
		}

		qMap := map[string]interface{}{
			"id":            id.String(),
			"question_text": qText,
			"question_type": qType,
		}
		if options != nil {
			qMap["options"] = json.RawMessage(*options)
		} else {
			qMap["options"] = json.RawMessage(`[]`)
		}
		questions = append(questions, qMap)
	}

	return questions, nil
}

func (s *QuizService) SaveQuizAnswer(ctx context.Context, attemptID string, userID string, questionID string, userAnswer string, timeSpent int, isSkipped bool) error {
	return nil
}

func (s *QuizService) SubmitQuizAttempt(ctx context.Context, attemptID string, userID string, tags []string) error {
	return nil
}
