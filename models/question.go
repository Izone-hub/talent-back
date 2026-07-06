package models

import (
	"encoding/json"
	"fmt"
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
	ID               uuid.UUID          `json:"id" db:"id"`
	QuestionText     string             `json:"question_text" db:"question_text"`
	QuestionType     QuestionType       `json:"question_type" db:"question_type"`
	Difficulty       QuestionDifficulty `json:"difficulty" db:"difficulty"`
	Options          json.RawMessage    `json:"options,omitempty" db:"options"`
	CorrectAnswer    *string            `json:"correct_answer,omitempty" db:"correct_answer"`
	Explanation      *string            `json:"explanation,omitempty" db:"explanation"`
	TimeLimitSeconds int                `json:"time_limit_seconds" db:"time_limit_seconds"`
	Points           int                `json:"points" db:"points"`
	Tags             []string           `json:"tags" db:"tags"`
	CreatedBy        *uuid.UUID         `json:"created_by,omitempty" db:"created_by"`
	IsActive         bool               `json:"is_active" db:"is_active"`
	UsageCount       int                `json:"usage_count" db:"usage_count"`
	CreatedAt        time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at" db:"updated_at"`
}

// CodingQuestion holds extra data for coding challenge questions.
type CodingQuestion struct {
	ID                 uuid.UUID       `json:"id" db:"id"`
	QuestionID         uuid.UUID       `json:"question_id" db:"question_id"`
	Language           string          `json:"language" db:"language"`
	CodeTemplate       *string         `json:"code_template,omitempty" db:"code_template"`
	TestCases          json.RawMessage `json:"test_cases" db:"test_cases"`
	ExecutionTimeLimit int             `json:"execution_time_limit" db:"execution_time_limit"`
	MemoryLimit        int             `json:"memory_limit" db:"memory_limit"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
}

// QuestionTag is the junction table for questions and tags.
type QuestionTag struct {
	QuestionID uuid.UUID `json:"question_id" db:"question_id"`
	TagID      uuid.UUID `json:"tag_id" db:"tag_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// CreateQuestionRequest is the payload for creating a new question.
type CreateQuestionRequest struct {
	QuestionText     string               `json:"question_text" validate:"required"`
	QuestionType     QuestionType         `json:"question_type" validate:"required"`
	Difficulty       QuestionDifficulty   `json:"difficulty" validate:"required"`
	Options          json.RawMessage      `json:"options,omitempty"`
	CorrectAnswer    *string              `json:"correct_answer,omitempty"`
	Explanation      *string              `json:"explanation,omitempty"`
	TimeLimitSeconds int                  `json:"time_limit_seconds"`
	Points           int                  `json:"points"`
	Tags             []string             `json:"tags"`
	CodingDetails    *CreateCodingDetails `json:"coding_details,omitempty"`
}

// CreateCodingDetails holds specific data for coding challenges.
type CreateCodingDetails struct {
	Language           string          `json:"language" validate:"required"`
	CodeTemplate       *string         `json:"code_template,omitempty"`
	TestCases          json.RawMessage `json:"test_cases" validate:"required"`
	ExecutionTimeLimit int             `json:"execution_time_limit"`
	MemoryLimit        int             `json:"memory_limit"`
}

// UpdateQuestionRequest is the payload for updating an existing question.
type UpdateQuestionRequest struct {
	QuestionText     *string              `json:"question_text,omitempty"`
	Options          json.RawMessage      `json:"options,omitempty"`
	CorrectAnswer    *string              `json:"correct_answer,omitempty"`
	Explanation      *string              `json:"explanation,omitempty"`
	TimeLimitSeconds *int                 `json:"time_limit_seconds,omitempty"`
	Points           *int                 `json:"points,omitempty"`
	Tags             []string             `json:"tags,omitempty"`
	CodingDetails    *UpdateCodingDetails `json:"coding_details,omitempty"`
}

// UpdateCodingDetails holds specific data for updating coding challenges.
type UpdateCodingDetails struct {
	Language           *string         `json:"language,omitempty"`
	CodeTemplate       *string         `json:"code_template,omitempty"`
	TestCases          json.RawMessage `json:"test_cases,omitempty"`
	ExecutionTimeLimit *int            `json:"execution_time_limit,omitempty"`
	MemoryLimit        *int            `json:"memory_limit,omitempty"`
}

// QuestionResponse is a combined view of a question and its details.
type QuestionResponse struct {
	Question
	CodingDetails *CodingQuestion `json:"coding_details,omitempty"`
}

func (r *CreateQuestionRequest) Validate() error {
	if r.QuestionType == QuestionTypeCodingChallenge && r.CodingDetails == nil {
		return fmt.Errorf("coding_details is required for coding_challenge questions")
	}
	return nil
}

func (r *UpdateQuestionRequest) Validate() error {
	// Basic validation logic
	return nil
}
