package models

import (
	"time"

	"github.com/google/uuid"
)

type QuizResults struct {
	ID            uuid.UUID `json:"id" db:"id"`
	QuizAttemptID uuid.UUID `json:"quiz_attempt_id" db:"quiz_attempt_id"`

	EasyCorrect   int `json:"easy_correct" db:"easy_correct"`
	EasyTotal     int `json:"easy_total" db:"easy_total"`
	MediumCorrect int `json:"medium_correct" db:"medium_correct"`
	MediumTotal   int `json:"medium_total" db:"medium_total"`
	HardCorrect   int `json:"hard_correct" db:"hard_correct"`
	HardTotal     int `json:"hard_total" db:"hard_total"`
	ExpertCorrect int `json:"expert_correct" db:"expert_correct"`
	ExpertTotal   int `json:"expert_total" db:"expert_total"`

	MultipleChoiceCorrect int `json:"multiple_choice_correct" db:"multiple_choice_correct"`
	MultipleChoiceTotal   int `json:"multiple_choice_total" db:"multiple_choice_total"`
	CodingCorrect         int `json:"coding_correct" db:"coding_correct"`
	CodingTotal           int `json:"coding_total" db:"coding_total"`

	AvgTimePerQuestionSeconds  *float64 `json:"avg_time_per_question_seconds,omitempty" db:"avg_time_per_question_seconds"`
	FastestQuestionTimeSeconds *int     `json:"fastest_question_time_seconds,omitempty" db:"fastest_question_time_seconds"`
	SlowestQuestionTimeSeconds *int     `json:"slowest_question_time_seconds,omitempty" db:"slowest_question_time_seconds"`

	AIFeedback *string  `json:"ai_feedback,omitempty" db:"ai_feedback"`
	Strengths  []string `json:"strengths,omitempty" db:"strengths"`
	Weaknesses []string `json:"weaknesses,omitempty" db:"weaknesses"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
