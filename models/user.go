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
}

func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

func (u *User) SanitizedUser() map[string]interface{} {
	return map[string]interface{}{
		"id":              u.ID,
		"github_id":       u.GithubID,
		"github_username": u.GithubUsername,
		"email":           u.Email,
		"avatar_url":      u.AvatarURL,
		"name":            u.Name,
		"role":            u.Role,
		"last_login_at":   u.LastLoginAt,
		"created_at":      u.CreatedAt,
	}
}

// ToResponse returns a copy of the user safe for API responses (same struct; sensitive fields use json:"-" so are omitted).
func (u *User) ToResponse() User {
	return User{
		ID:             u.ID,
		GithubID:       u.GithubID,
		GithubUsername: u.GithubUsername,
		Email:          u.Email,
		AvatarURL:      u.AvatarURL,
		Name:           u.Name,
		Role:           u.Role,
		LastLoginAt:    u.LastLoginAt,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}
