package models

import "github.com/google/uuid"

type QuizResponse struct {
	ID             uuid.UUID `json:"id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	QuizType       string    `json:"type"`
	Difficulty     string    `json:"difficulty"`
	TimeLimit      string    `json:"time_limit"`
	QuestionsCount int       `json:"questions_count"`
	Points         int       `json:"points"`
	Status         string    `json:"status"`
	Score          string    `json:"score,omitempty"`
}
