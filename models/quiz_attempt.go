package models

import (
	"time"

	"github.com/google/uuid"
)

// QuizAttemptStatus represents the status of a quiz attempt.
type QuizAttemptStatus string

const (
	QuizAttemptStatusStarted   QuizAttemptStatus = "started"
	QuizAttemptStatusInProgress QuizAttemptStatus = "in_progress"
	QuizAttemptStatusPaused    QuizAttemptStatus = "paused"
	QuizAttemptStatusCompleted QuizAttemptStatus = "completed"
	QuizAttemptStatusTimedOut  QuizAttemptStatus = "timed_out"
	QuizAttemptStatusAbandoned QuizAttemptStatus = "abandoned"
)

type QuizAttempt struct {
	ID            uuid.UUID         `json:"id" db:"id"`
	ApplicationID uuid.UUID         `json:"application_id" db:"application_id"`
	UserID        uuid.UUID         `json:"user_id" db:"user_id"`
	JobID         uuid.UUID         `json:"job_id" db:"job_id"`

	TotalQuestions    int  `json:"total_questions" db:"total_questions"`
	QuestionsPerQuiz  int  `json:"questions_per_quiz" db:"questions_per_quiz"`
	TimeLimitMinutes  *int `json:"time_limit_minutes,omitempty" db:"time_limit_minutes"`
	PassingScore      int  `json:"passing_score" db:"passing_score"`

	Status         QuizAttemptStatus `json:"status" db:"status"`
	StartedAt      time.Time         `json:"started_at" db:"started_at"`
	CompletedAt    *time.Time        `json:"completed_at,omitempty" db:"completed_at"`
	LastActivityAt time.Time         `json:"last_activity_at" db:"last_activity_at"`

	Score             *int  `json:"score,omitempty" db:"score"`
	CorrectAnswers    int   `json:"correct_answers" db:"correct_answers"`
	IncorrectAnswers  int   `json:"incorrect_answers" db:"incorrect_answers"`
	SkippedQuestions  int   `json:"skipped_questions" db:"skipped_questions"`
	Passed            *bool `json:"passed,omitempty" db:"passed"`

	TimeSpentSeconds         int `json:"time_spent_seconds" db:"time_spent_seconds"`
	AutoSaveIntervalSeconds  int `json:"auto_save_interval_seconds" db:"auto_save_interval_seconds"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
