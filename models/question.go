package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// QuestionType represents the type of quiz question.
type QuestionType string

const (
	QuestionTypeMultipleChoice  QuestionType = "multiple_choice"
	QuestionTypeMultipleSelect  QuestionType = "multiple_select"
	QuestionTypeText            QuestionType = "text"
	QuestionTypeTrueFalse       QuestionType = "true_false"
	QuestionTypeCodingChallenge QuestionType = "coding_challenge"
)

// QuestionDifficulty represents the difficulty level.
type QuestionDifficulty string

const (
	QuestionDifficultyEasy   QuestionDifficulty = "easy"
	QuestionDifficultyMedium QuestionDifficulty = "medium"
	QuestionDifficultyHard   QuestionDifficulty = "hard"
	QuestionDifficultyExpert QuestionDifficulty = "expert"
)

type Question struct {
	ID             uuid.UUID          `json:"id" db:"id"`
	QuestionText   string            `json:"question_text" db:"question_text"`
	QuestionType   QuestionType      `json:"question_type" db:"question_type"`
	Difficulty     QuestionDifficulty `json:"difficulty" db:"difficulty"`
	Options        json.RawMessage   `json:"options,omitempty" db:"options"`
	CorrectAnswer  *string           `json:"correct_answer,omitempty" db:"correct_answer"`
	Explanation    *string           `json:"explanation,omitempty" db:"explanation"`
	TimeLimitSeconds int             `json:"time_limit_seconds" db:"time_limit_seconds"`
	Points         int               `json:"points" db:"points"`
	Tags           []string         `json:"tags" db:"tags"`
	CreatedBy      *uuid.UUID        `json:"created_by,omitempty" db:"created_by"`
	IsActive       bool              `json:"is_active" db:"is_active"`
	UsageCount     int               `json:"usage_count" db:"usage_count"`
	CreatedAt      time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at" db:"updated_at"`
}

// CodingQuestion holds extra data for coding challenge questions.
type CodingQuestion struct {
	ID                   uuid.UUID      `json:"id" db:"id"`
	QuestionID           uuid.UUID      `json:"question_id" db:"question_id"`
	Language             string         `json:"language" db:"language"`
	CodeTemplate         *string        `json:"code_template,omitempty" db:"code_template"`
	TestCases            json.RawMessage `json:"test_cases" db:"test_cases"`
	ExecutionTimeLimit   int            `json:"execution_time_limit" db:"execution_time_limit"`
	MemoryLimit          int            `json:"memory_limit" db:"memory_limit"`
	CreatedAt            time.Time      `json:"created_at" db:"created_at"`
}

// QuestionTag is the junction table for questions and tags.
type QuestionTag struct {
	QuestionID uuid.UUID `json:"question_id" db:"question_id"`
	TagID      uuid.UUID `json:"tag_id" db:"tag_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}
