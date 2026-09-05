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

	// New developer fields
	PublicRepos       int        `json:"public_repos" db:"public_repos"`
	PublicGists       int        `json:"public_gists" db:"public_gists"`
	Followers         int        `json:"followers" db:"followers"`
	Following         int        `json:"following" db:"following"`
	Hireable          bool       `json:"hireable" db:"hireable"`
	Blog              *string    `json:"blog,omitempty" db:"blog"`
	Company           *string    `json:"company,omitempty" db:"company"`
	Location          *string    `json:"location,omitempty" db:"location"`
	Bio               *string    `json:"bio,omitempty" db:"bio"`
	TwitterUsername   *string    `json:"twitter_username,omitempty" db:"twitter_username"`
	TopLanguages      []string   `json:"top_languages" db:"top_languages"`
	ContributionCount int        `json:"contribution_count" db:"contribution_count"`
	AcceptanceJobID   *uuid.UUID `json:"acceptance_job_id,omitempty" db:"acceptance_job_id"`
}

func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// ToResponse returns a copy of the user safe for API responses
func (u *User) ToResponse() User {
	return User{
		ID:                u.ID,
		GithubID:          u.GithubID,
		GithubUsername:    u.GithubUsername,
		Email:             u.Email,
		AvatarURL:         u.AvatarURL,
		Name:              u.Name,
		Role:              u.Role,
		LastLoginAt:       u.LastLoginAt,
		CreatedAt:         u.CreatedAt,
		UpdatedAt:         u.UpdatedAt,
		PublicRepos:       u.PublicRepos,
		PublicGists:       u.PublicGists,
		Followers:         u.Followers,
		Following:         u.Following,
		Hireable:          u.Hireable,
		Blog:              u.Blog,
		Company:           u.Company,
		Location:          u.Location,
		Bio:               u.Bio,
		TwitterUsername:   u.TwitterUsername,
		TopLanguages:      u.TopLanguages,
		ContributionCount: u.ContributionCount,
		AcceptanceJobID:   u.AcceptanceJobID,
	}
}
