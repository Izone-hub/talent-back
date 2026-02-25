package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type GitHubService struct {
	clientID     string
	clientSecret string
	redirectURL  string
	frontendURL  string
}

type GitHubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	Bio       string `json:"bio"`
	Location  string `json:"location"`
	Company   string `json:"company"`
	Blog      string `json:"blog"`
	HTMLURL   string `json:"html_url"`
}

type GitHubRepo struct {
	ID          int64             `json:"id"`
	Name        string            `json:"name"`
	FullName    string            `json:"full_name"`
	Description string            `json:"description"`
	URL         string            `json:"url"`
	HTMLURL     string            `json:"html_url"`
	Language    string            `json:"language"`
	Languages   map[string]int    `json:"-"`
	Stars       int               `json:"stargazers_count"`
	Forks       int               `json:"forks_count"`
	Private     bool              `json:"private"`
	Readme      string            `json:"-"`
	ReadmeHTML  string            `json:"-"`
	TechStack   map[string]string `json:"-"`
}

func NewGitHubService(clientID, clientSecret, redirectURL, frontendURL string) *GitHubService {
	return &GitHubService{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		frontendURL:  frontendURL,
	}
}

func (gs *GitHubService) GetClientID() string {
	return gs.clientID
}

func (gs *GitHubService) GetRedirectURL() string {
	return gs.redirectURL
}

func (gs *GitHubService) GetFrontendURL() string {
	return gs.frontendURL
}

func (gs *GitHubService) GetAccessToken(code string) (string, error) {
	url := "https://github.com/login/oauth/access_token"

	reqBody := fmt.Sprintf(
		"client_id=%s&client_secret=%s&code=%s",
		gs.clientID, gs.clientSecret, code,
	)

	req, err := http.NewRequest("POST", url, strings.NewReader(reqBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("GitHub OAuth error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	return tokenResp.AccessToken, nil
}

func (gs *GitHubService) GetUser(accessToken string) (*GitHubUser, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s", string(body))
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (gs *GitHubService) GetUserRepos(accessToken string) ([]*GitHubRepo, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/repos?per_page=100&sort=updated", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s", string(body))
	}

	var repos []*GitHubRepo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}

	// Fetch additional data for each repo
	for _, repo := range repos {
		// Fetch languages
		gs.fetchRepoLanguages(accessToken, repo)

		// Fetch README
		gs.fetchRepoReadme(accessToken, repo)

		// Extract tech stack
		gs.extractTechStack(accessToken, repo)
	}

	return repos, nil
}

func (gs *GitHubService) fetchRepoLanguages(accessToken string, repo *GitHubRepo) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/languages", repo.FullName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var languages map[string]int
		if err := json.NewDecoder(resp.Body).Decode(&languages); err == nil {
			repo.Languages = languages
		}
	}

	return nil
}

func (gs *GitHubService) fetchRepoReadme(accessToken string, repo *GitHubRepo) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/readme", repo.FullName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var readmeResp struct {
			Content  string `json:"content"`
			Encoding string `json:"encoding"`
			HTMLURL  string `json:"html_url"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&readmeResp); err == nil {
			if readmeResp.Encoding == "base64" {
				decoded, err := base64.StdEncoding.DecodeString(readmeResp.Content)
				if err == nil {
					repo.Readme = string(decoded)
				}
			}
			repo.ReadmeHTML = readmeResp.HTMLURL
		}
	}

	return nil
}

func (gs *GitHubService) extractTechStack(accessToken string, repo *GitHubRepo) {
	techStack := make(map[string]string)

	// Check for common dependency files
	files := []string{"package.json", "go.mod", "requirements.txt", "pom.xml", "Gemfile", "Cargo.toml", "composer.json"}

	for _, file := range files {
		url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", repo.FullName, file)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var fileResp struct {
				Content  string `json:"content"`
				Encoding string `json:"encoding"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&fileResp); err == nil {
				if fileResp.Encoding == "base64" {
					decoded, err := base64.StdEncoding.DecodeString(fileResp.Content)
					if err == nil {
						content := string(decoded)
						extracted := ExtractTechStackFromFile(file, content)
						for k, v := range extracted {
							techStack[k] = v
						}
					}
				}
			}
		}
		resp.Body.Close()
	}

	repo.TechStack = techStack
}
