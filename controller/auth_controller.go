package controller

import (
	"encoding/json"
	"net/http"

	"github.com/Izone-hub/talent-backend/service"
)

type AuthController struct {
	authService *service.AuthService
}

func NewAuthController(authService *service.AuthService) *AuthController {
	return &AuthController{
		authService: authService,
	}
}

// GitHubLogin initiates GitHub OAuth flow
func (c *AuthController) GitHubLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, c.authService.GitHubAuthURL(), http.StatusTemporaryRedirect)
}

// GitHubCallback handles the OAuth callback from GitHub
func (c *AuthController) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	// Get the code from query parameters
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not provided", http.StatusBadRequest)
		return
	}

	// Handle the callback through auth service
	authResponse, err := c.authService.HandleGitHubCallback(r.Context(), code)
	if err != nil {
		http.Error(w, "Authentication failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the token and user info as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authResponse)
}

// GetCurrentUser returns the currently authenticated user
func (c *AuthController) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Return user info
	response := map[string]interface{}{
		"user_id":         claims.UserID,
		"github_id":       claims.GithubID,
		"github_username": claims.GithubUsername,
		"role":            claims.Role,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
