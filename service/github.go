package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/Izone-hub/talent-backend/config"
)

type GithubService struct {
	config     *config.Config
	httpClient *http.Client
}

// Enhanced GitHubUser struct with all the fields we need
type GitHubUser struct {
	ID              int64  `json:"id"`
	Login           string `json:"login"`
	AvatarURL       string `json:"avatar_url"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	PublicRepos     int    `json:"public_repos"`
	PublicGists     int    `json:"public_gists"`
	Followers       int    `json:"followers"`
	Following       int    `json:"following"`
	Hireable        bool   `json:"hireable"`
	Blog            string `json:"blog"`
	Company         string `json:"company"`
	Location        string `json:"location"`
	Bio             string `json:"bio"`
	TwitterUsername string `json:"twitter_username"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// GitHubRepo represents a repository from GitHub API
type GitHubRepo struct {
	Name            string `json:"name"`
	FullName        string `json:"full_name"`
	Description     string `json:"description"`
	Language        string `json:"language"`
	Private         bool   `json:"private"`
	StargazersCount int    `json:"stargazers_count"`
	ForksCount      int    `json:"forks_count"`
	WatchersCount   int    `json:"watchers_count"`
	HTMLURL         string `json:"html_url"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	PushedAt        string `json:"pushed_at"`
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
	tokenURL := fmt.Sprintf("https://github.com/login/oauth/access_token?client_id=%s&client_secret=%s&code=%s",
		s.config.GitHubClientID, s.config.GitHubClientSecret, code)

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

// GetUser fetches GitHub user info using access token - Enhanced version
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	var githubUser GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}

	// If email is private, it might be empty. Fetch from /user/emails
	if githubUser.Email == "" {
		fmt.Println("GitHub email is empty from /user endpoint. Attempting to fetch from /user/emails...")
		emailReq, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
		if err == nil {
			emailReq.Header.Set("Authorization", "Bearer "+accessToken)
			emailReq.Header.Set("Accept", "application/vnd.github.v3+json")

			emailResp, err := s.httpClient.Do(emailReq)
			if err != nil {
				fmt.Printf("Error fetching emails: %v\n", err)
			} else {
				fmt.Printf("GitHub /user/emails returned status: %d\n", emailResp.StatusCode)
				if emailResp.StatusCode == http.StatusOK {
					type GitHubEmail struct {
						Email    string `json:"email"`
						Primary  bool   `json:"primary"`
						Verified bool   `json:"verified"`
					}
					var emails []GitHubEmail
					if err := json.NewDecoder(emailResp.Body).Decode(&emails); err == nil {
						fmt.Printf("Decoded emails: %+v\n", emails)
						for _, e := range emails {
							if e.Primary && e.Verified {
								githubUser.Email = e.Email
								break
							}
						}
						// Fallback to any verified email if primary isn't found
						if githubUser.Email == "" {
							for _, e := range emails {
								if e.Verified {
									githubUser.Email = e.Email
									break
								}
							}
						}
						// Fallback to any email at all
						if githubUser.Email == "" && len(emails) > 0 {
							githubUser.Email = emails[0].Email
						}
						fmt.Printf("Selected email: %s\n", githubUser.Email)
					} else {
						fmt.Printf("Failed to decode emails: %v\n", err)
					}
				}
				emailResp.Body.Close()
			}
		} else {
			fmt.Printf("Failed to create email request: %v\n", err)
		}
	}

	return &githubUser, nil
}

// GetUserRepos fetches the user's repositories from GitHub
func (s *GithubService) GetUserRepos(ctx context.Context, accessToken, username string) ([]GitHubRepo, error) {
	// Get public repositories (you can also include private if needed)
	url := fmt.Sprintf("https://api.github.com/users/%s/repos?sort=updated&per_page=100&type=public", username)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create repos request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user repos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// If rate limited or other error, return empty slice but don't fail
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
			return []GitHubRepo{}, fmt.Errorf("rate limited by GitHub API")
		}
		return nil, fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	var repos []GitHubRepo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, fmt.Errorf("failed to decode repos response: %w", err)
	}

	return repos, nil
}

// GetUserStarredRepos fetches repositories starred by the user
func (s *GithubService) GetUserStarredRepos(ctx context.Context, accessToken, username string) ([]GitHubRepo, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s/starred?per_page=100", username)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create starred repos request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get starred repos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []GitHubRepo{}, nil // Don't fail on starred repos
	}

	var repos []GitHubRepo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}

	return repos, nil
}

// CalculateTopLanguages analyzes repos and returns top 5 languages
func (s *GithubService) CalculateTopLanguages(repos []GitHubRepo) []string {
	langCount := make(map[string]int)

	for _, repo := range repos {
		if repo.Language != "" && !repo.Private {
			langCount[repo.Language]++
		}
	}

	// Convert to slice for sorting
	type langFreq struct {
		Name  string
		Count int
	}

	freqs := make([]langFreq, 0, len(langCount))
	for name, count := range langCount {
		freqs = append(freqs, langFreq{name, count})
	}

	// Sort by count descending
	sort.Slice(freqs, func(i, j int) bool {
		return freqs[i].Count > freqs[j].Count
	})

	// Take top 5
	result := make([]string, 0, 5)
	for i := 0; i < len(freqs) && i < 5; i++ {
		result = append(result, freqs[i].Name)
	}

	return result
}

// GetUserWithDetails fetches user info and repos in one call (for convenience)
func (s *GithubService) GetUserWithDetails(ctx context.Context, accessToken string) (*GitHubUser, []GitHubRepo, error) {
	// Get user info first
	user, err := s.GetUser(ctx, accessToken)
	if err != nil {
		return nil, nil, err
	}

	// Then get their repos
	repos, err := s.GetUserRepos(ctx, accessToken, user.Login)
	if err != nil {
		// Log error but don't fail - we can still return user without repos
		fmt.Printf("Warning: Failed to fetch repos for %s: %v\n", user.Login, err)
		repos = []GitHubRepo{}
	}

	return user, repos, nil
}
