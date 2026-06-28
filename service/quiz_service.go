package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// GetUserQuizzes retrieves all quiz attempts belonging to a specific user
func (s *QuizService) GetUserQuizzes(ctx context.Context, userID string) ([]QuizAttempt, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT qa.id, qa.user_id, COALESCE(j.title, 'Quiz') as title, qa.status, qa.created_at
		FROM quiz_attempts qa
		LEFT JOIN jobs j ON qa.job_id = j.id
		WHERE qa.user_id = $1
		ORDER BY qa.created_at DESC
	`, userUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quizzes []QuizAttempt
	for rows.Next() {
		var q QuizAttempt
		if err := rows.Scan(&q.ID, &q.UserID, &q.Title, &q.Status, &q.CreatedAt); err != nil {
			return nil, err
		}
		quizzes = append(quizzes, q)
	}

	if quizzes == nil {
		quizzes = []QuizAttempt{}
	}

	return quizzes, nil
}

// GetQuizAttempt fetches the profile configuration of a specific attempt
func (s *QuizService) GetQuizAttempt(ctx context.Context, attemptID string, userID string) (*QuizAttempt, error) {
	attUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return nil, err
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	return &QuizAttempt{
		ID:     attUUID,
		UserID: userUUID,
		Title:  "Technical Core Assessment",
		Type:   "go,svelte,postgresql",
		Status: "active",
	}, nil
}

// StartQuizAttempt initializes an active session entry point in the actual database
// Updated signature to accept external IDs
func (s *QuizService) StartQuizAttempt(ctx context.Context, attemptID, userID, appID, jobID string) error {
	// 1. Check if an active attempt exists for this application
	var existingID string
	err := s.pool.QueryRow(ctx,
		"SELECT id FROM quiz_attempts WHERE application_id = $1 AND status = 'started'",
		appID).Scan(&existingID)

	if err == nil {
		// Attempt already exists! Return nil (success) or handle as "Resume"
		log.Println("Active attempt already exists, resuming:", existingID)
		return nil
	}

	// 2. Only proceed with INSERT if no attempt was found
	attemptUUID := uuid.MustParse(attemptID)
	appUUID := uuid.MustParse(appID)
	userUUID := uuid.MustParse(userID)
	jobUUID := uuid.MustParse(jobID)

	query := `
        INSERT INTO quiz_attempts (
            id, application_id, user_id, job_id, total_questions, 
            questions_per_quiz, passing_score, status, 
            started_at, last_activity_at, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, 10, 10, 70, 'started', NOW(), NOW(), NOW(), NOW())
    `

	_, err = s.pool.Exec(ctx, query, attemptUUID, appUUID, userUUID, jobUUID)
	return err
}

func (s *QuizService) GetNextQuestion(ctx context.Context, attemptID string, userID string) (map[string]interface{}, error) {
	attemptUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return nil, err
	}

	// Get quiz attempt config: questions_per_quiz and count answered so far
	var questionsPerQuiz int
	var answeredCount int
	err = s.pool.QueryRow(ctx, `
        SELECT qp.questions_per_quiz,
               (SELECT COUNT(*) FROM quiz_answers qa WHERE qa.quiz_attempt_id = qp.id)
        FROM quiz_attempts qp
        WHERE qp.id = $1
    `, attemptUUID).Scan(&questionsPerQuiz, &answeredCount)
	if err != nil {
		log.Printf("GetNextQuestion ERROR fetching attempt %s: %v", attemptID, err)
		return nil, err
	}

	// If already answered questions_per_quiz questions, quiz is finished
	if answeredCount >= questionsPerQuiz {
		return map[string]interface{}{
			"status":  "finished",
			"message": "You have answered all questions in this quiz",
		}, nil
	}

	query := `
        SELECT q.id, q.question_text, q.question_type, q.options, q.correct_answer, q.difficulty
        FROM questions q
        WHERE NOT EXISTS (
            SELECT 1 FROM quiz_answers qa 
            WHERE qa.question_id = q.id AND qa.quiz_attempt_id = $1
        )
        ORDER BY RANDOM()
        LIMIT 1;
    `

	var id uuid.UUID
	var qText, qType, difficulty string
	var options, correctAnswer *string

	err = s.pool.QueryRow(ctx, query, attemptUUID).Scan(&id, &qText, &qType, &options, &correctAnswer, &difficulty)
	if err != nil {
		log.Printf("GetNextQuestion ERROR for attemptID=%s: %v", attemptID, err)
		return nil, err
	}

	qMap := map[string]interface{}{
		"id":            id.String(),
		"question_text": qText,
		"question_type": qType,
		"difficulty":    difficulty,
	}

	if options != nil {
		qMap["options"] = json.RawMessage(*options)
	} else {
		qMap["options"] = json.RawMessage(`[]`)
	}

	return qMap, nil
}

// SaveQuizAnswer updates the database state by cataloging the submitted single question response
// Inside service/quiz_service.go

func (s *QuizService) SaveQuizAnswer(ctx context.Context, attemptID, userID, questionID, answer string, timeSpent int, isSkipped bool) error {
	log.Printf("DEBUG: Validating and saving answer for QuestionID: %s", questionID)

	// 1. Convert strings to UUIDs
	attemptUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return fmt.Errorf("invalid attempt ID: %w", err)
	}
	questionUUID, err := uuid.Parse(questionID)
	if err != nil {
		return fmt.Errorf("invalid question ID: %w", err)
	}

	// 2. Fetch the correct answer from the 'questions' table
	var dbCorrectAnswer *string
	err = s.pool.QueryRow(ctx, "SELECT correct_answer FROM questions WHERE id = $1", questionUUID).Scan(&dbCorrectAnswer)
	if err != nil {
		return fmt.Errorf("failed to fetch correct answer for validation: %w", err)
	}

	// 3. Determine if the user's answer is correct
	isCorrect := dbCorrectAnswer != nil && answer == *dbCorrectAnswer

	// 4. Perform the INSERT
	newAnswerID := uuid.New()

	// Note: I added 'is_correct' to the column list and the values ($7)
	query := `
        INSERT INTO quiz_answers (
            id, quiz_attempt_id, question_id, user_answer, time_spent_seconds, is_skipped, is_correct, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
    `

	_, err = s.pool.Exec(ctx, query, newAnswerID, attemptUUID, questionUUID, answer, timeSpent, isSkipped, isCorrect)

	return err
}

// SubmitQuizAttempt calculates and locks progress finality status safely
func (s *QuizService) SubmitQuizAttempt(ctx context.Context, attemptID string, userID string, tags []string) error {
	attemptUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return fmt.Errorf("invalid attempt ID: %w", err)
	}

	// Check if already completed
	var currentStatus string
	err = s.pool.QueryRow(ctx, "SELECT status FROM quiz_attempts WHERE id = $1", attemptUUID).Scan(&currentStatus)
	if err != nil {
		return fmt.Errorf("could not find quiz attempt: %w", err)
	}
	if currentStatus == "completed" {
		return fmt.Errorf("quiz attempt %s is already completed", attemptID)
	}

	// Start a database transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Get the APPLICATION_ID associated with this quiz
	var appID uuid.UUID
	err = tx.QueryRow(ctx, "SELECT application_id FROM quiz_attempts WHERE id = $1", attemptUUID).Scan(&appID)
	if err != nil {
		return fmt.Errorf("could not find application_id for this quiz attempt: %w", err)
	}

	// 2. Calculate score
	var correctCount int
	var totalQuestions int
	err = tx.QueryRow(ctx,
		"SELECT COUNT(*) FILTER (WHERE is_correct = true), COUNT(*) FROM quiz_answers WHERE quiz_attempt_id = $1",
		attemptUUID).Scan(&correctCount, &totalQuestions)
	if err != nil {
		return err
	}

	score := 0.0
	if totalQuestions > 0 {
		score = (float64(correctCount) / float64(totalQuestions)) * 100
	}

	// 3. Update Quiz status to completed
	_, err = tx.Exec(ctx,
		"UPDATE quiz_attempts SET status = 'completed', score = $2, correct_answers = $3, updated_at = NOW() WHERE id = $1",
		attemptUUID, int(score), correctCount)
	if err != nil {
		return err
	}

	// 4. Update the actual application status
	_, err = tx.Exec(ctx, `
        UPDATE job_applications 
        SET status = 'quiz_completed', 
            quiz_score = $1, 
            quiz_completed_at = NOW() 
        WHERE id = $2`,
		score, appID)
	if err != nil {
		return fmt.Errorf("failed to update job_applications table: %w", err)
	}

	return tx.Commit(ctx)
}
