package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Izone-hub/talent-backend/config"
	"github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type AuthService struct {
	config        *config.Config
	githubService *GithubService
	queries       *database.Queries
}

// JWT Payload. this defines what data is stored in the JWT token.
type Claims struct {
	UserID         uuid.UUID `json:"user_id"`
	GithubID       int64     `json:"github_id"`
	GithubUsername string    `json:"github_username"`
	ApplicationID  uuid.UUID `json:"application_id"` // Add this
	JobID          uuid.UUID `json:"job_id"`         // Add this
	Role           string    `json:"role"`
	jwt.RegisteredClaims
}

type AuthResponse struct {
	Token string              `json:"token"`
	User  models.User         `json:"user"`
	Repos []models.Repository `json:"repos"`
}

func NewAuthService(cfg *config.Config, githubService *GithubService, db database.DBTX) *AuthService {
	return &AuthService{
		config:        cfg,
		githubService: githubService,
		queries:       database.New(db),
	}
}

// GitHubAuthURL returns the GitHub OAuth authorization URL for the login redirect.
// Exposes config needed for the OAuth flow without exporting the whole config.
func (s *AuthService) GitHubAuthURL() string {
	return "https://github.com/login/oauth/authorize?" +
		"client_id=" + s.config.GitHubClientID +
		"&redirect_uri=" + s.config.GitHubRedirectURL +
		"&scope=user:email"
}

// HandleGitHubCallback processes GitHub OAuth callback
func (s *AuthService) HandleGitHubCallback(ctx context.Context, code string) (*AuthResponse, error) {
	// 1. Exchange code for GitHub access token
	tokenResponse, err := s.githubService.ExchangeCodeForToken(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// 2. Get GitHub user info AND repositories using the new method
	githubUser, repos, err := s.githubService.GetUserWithDetails(ctx, tokenResponse.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub user: %w", err)
	}

	// 3. Calculate top languages from repositories
	topLanguages := s.githubService.CalculateTopLanguages(repos)

	// 4. Upsert user with ALL GitHub data (including repos and languages)
	dbUser, err := s.upsertUserFromGitHub(ctx, githubUser, tokenResponse, topLanguages)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert user: %w", err)
	}

	// 5. If user is allowed by the admin allowlist or configured admin list,
	// promote them and ensure an admin row exists.
	allowlisted, err := s.queries.IsGitHubAllowlisted(ctx, githubUser.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check admin allowlist: %w", err)
	}

	isConfiguredAdmin := false
	for _, admin := range s.config.GetAdminUsernames() {
		if admin == githubUser.Login {
			isConfiguredAdmin = true
			break
		}
	}

	if allowlisted || isConfiguredAdmin {
		if dbUser.Role != "admin" {
			dbUser, err = s.queries.UpdateUserRole(ctx, database.UpdateUserRoleParams{
				GithubID: githubUser.ID,
				Role:     "admin",
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update user role: %w", err)
			}
		}

		adminPermissions, err := json.Marshal(map[string]bool{
			"manage_jobs":              true,
			"manage_questions":         true,
			"manage_tags":              true,
			"manage_weights":           true,
			"manage_difficulties":      true,
			"view_applicants":          true,
			"view_cvs":                 true,
			"view_github_metadata":     true,
			"view_ai_summaries":        true,
			"view_applicant_summaries": true,
			"view_audit_logs":          true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to marshal admin permissions: %w", err)
		}

		_, err = s.queries.UpsertAdmin(ctx, database.UpsertAdminParams{
			UserID:      pgUUIDToUUID(dbUser.ID),
			GithubID:    pgtype.Int8{Int64: githubUser.ID, Valid: true},
			GithubLogin: pgtype.Text{String: githubUser.Login, Valid: true},
			Email:       strToPgText(githubUser.Email),
			Role:        "admin",
			Permissions: adminPermissions,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create/update admin record: %w", err)
		}
	}

	user := dbUserToModel(dbUser)

	// 6. Convert GitHub repos to response models
	repoModels := githubReposToModels(repos)

	// 7. Generate JWT token
	token, err := s.generateJWT(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthResponse{
		Token: token,
		User:  user.ToResponse(),
		Repos: repoModels,
	}, nil
}

// upsertUserFromGitHub creates or updates a user from GitHub OAuth data.
// Profile fields (public_repos, blog, etc.) are not stored; extend the users schema and use CreateOrUpdateUserWithProfile if needed.
// upsertUserFromGitHub creates or updates a user from GitHub OAuth data.
func (s *AuthService) upsertUserFromGitHub(ctx context.Context, githubUser *GitHubUser, tokenResponse *GitHubTokenResponse, topLanguages []string) (database.User, error) {
	expiresAt := time.Now().AddDate(1, 0, 0)

	if topLanguages == nil {
		topLanguages = []string{}
	}

	return s.queries.CreateOrUpdateUser(ctx, database.CreateOrUpdateUserParams{
		GithubID:             githubUser.ID,
		GithubUsername:       githubUser.Login,
		Email:                strToPgText(githubUser.Email),
		AvatarUrl:            strToPgText(githubUser.AvatarURL),
		Name:                 strToPgText(githubUser.Name),
		GithubAccessToken:    strToPgText(tokenResponse.AccessToken),
		GithubTokenExpiresAt: pgtype.Timestamp{Time: expiresAt, Valid: true},
		PublicRepos:          pgtype.Int4{Int32: int32(githubUser.PublicRepos), Valid: true},
		PublicGists:          pgtype.Int4{Int32: int32(githubUser.PublicGists), Valid: true},
		Followers:            pgtype.Int4{Int32: int32(githubUser.Followers), Valid: true},
		Following:            pgtype.Int4{Int32: int32(githubUser.Following), Valid: true},
		Hireable:             pgtype.Bool{Bool: githubUser.Hireable, Valid: true},
		Blog:                 strToPgText(githubUser.Blog),
		Company:              strToPgText(githubUser.Company),
		Location:             strToPgText(githubUser.Location),
		Bio:                  strToPgText(githubUser.Bio),
		TwitterUsername:      strToPgText(githubUser.TwitterUsername),
		TopLanguages:         topLanguages,
		ContributionCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
}

// dbUserToModel converts a database.User (pgtype fields) to models.User (standard types).
func dbUserToModel(u database.User) models.User {
	var id uuid.UUID
	if u.ID.Valid {
		id, _ = uuid.FromBytes(u.ID.Bytes[:])
	}

	var topLangs []string
	if u.TopLanguages != nil {
		topLangs = u.TopLanguages
	} else {
		topLangs = []string{}
	}

	return models.User{
		ID:                   id,
		GithubID:             u.GithubID,
		GithubUsername:       u.GithubUsername,
		Email:                pgTextToStrPtr(u.Email),
		AvatarURL:            pgTextToStrPtr(u.AvatarUrl),
		Name:                 pgTextToStrPtr(u.Name),
		Role:                 u.Role,
		GithubAccessToken:    pgTextToString(u.GithubAccessToken),
		GithubTokenExpiresAt: pgTimestampToTimePtr(u.GithubTokenExpiresAt),
		LastLoginAt:          pgTimestampToTime(u.LastLoginAt),
		CreatedAt:            pgTimestampToTime(u.CreatedAt),
		UpdatedAt:            pgTimestampToTime(u.UpdatedAt),
		PublicRepos:          int(u.PublicRepos.Int32),
		PublicGists:          int(u.PublicGists.Int32),
		Followers:            int(u.Followers.Int32),
		Following:            int(u.Following.Int32),
		Hireable:             u.Hireable.Bool,
		Blog:                 pgTextToStrPtr(u.Blog),
		Company:              pgTextToStrPtr(u.Company),
		Location:             pgTextToStrPtr(u.Location),
		Bio:                  pgTextToStrPtr(u.Bio),
		TwitterUsername:      pgTextToStrPtr(u.TwitterUsername),
		TopLanguages:         topLangs,
		ContributionCount:    int(u.ContributionCount.Int32),
	}
}

func (s *AuthService) generateJWT(user *models.User) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // Token expires in 24 hours

	claims := &Claims{
		UserID:         user.ID,
		GithubID:       user.GithubID,
		GithubUsername: user.GithubUsername,
		Role:           user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

// ValidateToken validates a JWT token and returns the claims (e.g. for auth middleware).
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWTSecret), nil
	})
	log.Println("token:", token)

	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// GetUserByID fetches a user by their UUID from the database.
func (s *AuthService) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	var pgID pgtype.UUID
	copy(pgID.Bytes[:], userID[:])
	pgID.Valid = true

	dbUser, err := s.queries.GetUserByID(ctx, pgID)
	if err != nil {
		return nil, err
	}

	user := dbUserToModel(dbUser)
	return &user, nil
}

// githubReposToModels converts GitHub API repo structs to response models.
func githubReposToModels(repos []GitHubRepo) []models.Repository {
	result := make([]models.Repository, 0, len(repos))
	for _, r := range repos {
		createdAt, _ := time.Parse(time.RFC3339, r.CreatedAt)
		updatedAt, _ := time.Parse(time.RFC3339, r.UpdatedAt)
		pushedAt, _ := time.Parse(time.RFC3339, r.PushedAt)

		desc := r.Description
		var descPtr *string
		if desc != "" {
			descPtr = &desc
		}

		result = append(result, models.Repository{
			Name:            r.Name,
			FullName:        r.FullName,
			Description:     descPtr,
			Language:        r.Language,
			Private:         r.Private,
			StargazersCount: r.StargazersCount,
			ForksCount:      r.ForksCount,
			WatchersCount:   r.WatchersCount,
			HTMLURL:         r.HTMLURL,
			CreatedAt:       createdAt,
			UpdatedAt:       updatedAt,
			PushedAt:        pushedAt,
		})
	}
	return result
}
