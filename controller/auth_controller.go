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

	// Redirect back to frontend with the token
	frontendURL := "http://localhost:5173/auth/callback?token=" + authResponse.Token
	http.Redirect(w, r, frontendURL, http.StatusTemporaryRedirect)
}

// GetCurrentUser returns the currently authenticated user
func (c *AuthController) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// 1. Get user claims from context (set by auth middleware)
	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Fetch full user details from database using the UserID in claims
	user, err := c.authService.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		http.Error(w, "Failed to fetch user data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Return sanitized user info (ToResponse excludes tokens)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user.ToResponse())
}
