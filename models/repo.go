package models

import (
	"time"
)

type Repository struct {
	Name            string    `json:"name"`
	FullName        string    `json:"full_name"`
	Description     *string   `json:"description,omitempty"`
	Language        string    `json:"language"`
	Private         bool      `json:"private"`
	StargazersCount int       `json:"stargazers_count"`
	ForksCount      int       `json:"forks_count"`
	WatchersCount   int       `json:"watchers_count"`
	HTMLURL         string    `json:"html_url"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	PushedAt        time.Time `json:"pushed_at"`
}

type RepositoryResponse struct {
	TotalCount   int          `json:"total_count"`
	Repositories []Repository `json:"repositories"`
	Languages    []string     `json:"languages,omitempty"` // For filtering
}
