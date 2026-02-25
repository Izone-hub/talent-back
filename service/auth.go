package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Izone-hub/talent-backend/config"
	"github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	Role           string    `json:"role"`
	jwt.RegisteredClaims
}

type AuthResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

func NewAuthService(cfg *config.Config, githubService *GithubService, db *pgx.Conn) *AuthService {
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

// HandleGitHubCallback processes GitHub OAuth callback: exchange code, fetch GitHub user,
// upsert our user (create or update token/last_login), then return JWT and user.
func (s *AuthService) HandleGitHubCallback(ctx context.Context, code string) (*AuthResponse, error) {
	// 1. Exchange code for GitHub access token
	tokenResponse, err := s.githubService.ExchangeCodeForToken(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// 2. Get GitHub user info
	githubUser, err := s.githubService.GetUser(ctx, tokenResponse.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub user: %w", err)
	}

	// 3. Upsert user (insert or update on conflict). SQL sets last_login_at = NOW() and updates token.
	dbUser, err := s.upsertUserFromGitHub(ctx, githubUser, tokenResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert user: %w", err)
	}

	// 4. If user is in admin list but not yet admin, promote (e.g. config changed or new user defaulted to 'user')
	for _, admin := range s.config.GetAdminUsernames() {
		if admin == githubUser.Login && dbUser.Role != "admin" {
			dbUser, err = s.queries.UpdateUserRole(ctx, database.UpdateUserRoleParams{GithubID: githubUser.ID, Role: "admin"})
			if err != nil {
				return nil, fmt.Errorf("failed to update user role: %w", err)
			}
			break
		}
	}

	user := dbUserToModel(dbUser)

	// 5. Generate JWT token
	token, err := s.generateJWT(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthResponse{
		Token: token,
		User:  user.ToResponse(),
	}, nil
}

// upsertUserFromGitHub inserts or updates a user from GitHub data (CreateOrUpdateUser upsert).
func (s *AuthService) upsertUserFromGitHub(ctx context.Context, githubUser *GitHubUser, tokenResponse *GitHubTokenResponse) (database.User, error) {
	expiresAt := time.Now().AddDate(1, 0, 0)
	return s.queries.CreateOrUpdateUser(ctx, database.CreateOrUpdateUserParams{
		GithubID:             githubUser.ID,
		GithubUsername:       githubUser.Login,
		Email:                strToPgText(githubUser.Email),
		AvatarUrl:            strToPgText(githubUser.AvatarURL),
		Name:                 strToPgText(githubUser.Name),
		GithubAccessToken:    strToPgText(tokenResponse.AccessToken),
		GithubTokenExpiresAt: pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
}

// dbUserToModel converts a database.User (pgtype fields) to models.User (standard types).
func dbUserToModel(u database.User) models.User {
	var id uuid.UUID
	if u.ID.Valid {
		id, _ = uuid.FromBytes(u.ID.Bytes[:])
	}
	return models.User{
		ID:             id,
		GithubID:       u.GithubID,
		GithubUsername: u.GithubUsername,
		Email:          pgTextToStrPtr(u.Email),
		AvatarURL:      pgTextToStrPtr(u.AvatarUrl),
		Name:           pgTextToStrPtr(u.Name),
		Role:           u.Role,
		GithubAccessToken: pgTextToString(u.GithubAccessToken),
		GithubTokenExpiresAt: pgTimestampToTimePtr(u.GithubTokenExpiresAt),
		LastLoginAt:   pgTimestampToTime(u.LastLoginAt),
		CreatedAt:     pgTimestampToTime(u.CreatedAt),
		UpdatedAt:     pgTimestampToTime(u.UpdatedAt),
	}
}

func strToPgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

func pgTextToStrPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func pgTextToString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func pgTimestampToTime(t pgtype.Timestamp) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func pgTimestampToTimePtr(t pgtype.Timestamp) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
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
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
