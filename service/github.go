package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Izone-hub/talent-backend/config"
)

type GithubService struct {
	config     *config.Config
	httpClient *http.Client
}

// data model these represents JSON response from GitHub API
type GitHubUser struct {
	ID          int64  `json:"id"`
	Login       string `json:"login"`
	AvatarURL   string `json:"avatar_url"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	PublicRepos int    `json:"public_repos"`
}

type GitHubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// error types
type GithubError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type GithubUnauthorizedError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// create a new service instance this is called when your app starts
func NewGithubService(cfg *config.Config) *GithubService {
	return &GithubService{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *GithubService) ExchangeCodeForToken(ctx context.Context, code string) (*GitHubTokenResponse, error) {
	// build the token URL
	tokenURL := fmt.Sprintf("https://github.com/login/oauth/access_token?client_id=%s&client_secret=%s&code=%s", s.config.GitHubClientID, s.config.GitHubClientSecret, code)

	// create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create the token request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	// send the request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to exchange code for token: %s", resp.Status)
	}

	// convert JSON to Go struct
	var tokenResponse GitHubTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResponse, nil

}

// GetUser fetches GitHub user info using access token
func (s *GithubService) GetUser(ctx context.Context, accessToken string) (*GitHubUser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var githubUser GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}

	return &githubUser, nil
}
