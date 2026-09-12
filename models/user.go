package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	GithubID             int64      `json:"github_id" db:"github_id"`
	GithubUsername       string     `json:"github_username" db:"github_username"`
	Email                *string    `json:"email,omitempty" db:"email"`
	AvatarURL            *string    `json:"avatar_url,omitempty" db:"avatar_url"`
	Name                 *string    `json:"name,omitempty" db:"name"`
	Role                 string     `json:"role" db:"role"`
	GithubAccessToken    string     `json:"-" db:"github_access_token"`
	GithubTokenExpiresAt *time.Time `json:"-" db:"github_token_expires_at"`
	LastLoginAt          time.Time  `json:"last_login_at" db:"last_login_at"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`

	// Developer profile fields
	PublicRepos     int        `json:"public_repos" db:"public_repos"`
	Followers       int        `json:"followers" db:"followers"`
	Following       int        `json:"following" db:"following"`
	Bio             *string    `json:"bio,omitempty" db:"bio"`
	TopLanguages    []string   `json:"top_languages" db:"top_languages"`
	AcceptanceJobID *uuid.UUID `json:"acceptance_job_id,omitempty" db:"acceptance_job_id"`
}

func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// ToResponse returns a copy of the user safe for API responses
func (u *User) ToResponse() User {
	return User{
		ID:              u.ID,
		GithubID:        u.GithubID,
		GithubUsername:  u.GithubUsername,
		Email:           u.Email,
		AvatarURL:       u.AvatarURL,
		Name:            u.Name,
		Role:            u.Role,
		LastLoginAt:     u.LastLoginAt,
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       u.UpdatedAt,
		PublicRepos:     u.PublicRepos,
		Followers:       u.Followers,
		Following:       u.Following,
		Bio:             u.Bio,
		TopLanguages:    u.TopLanguages,
		AcceptanceJobID: u.AcceptanceJobID,
	}
}
