package controller

import (
	"crypto/subtle"
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

// GitHubLogin initiates GitHub OAuth flow with CSRF state protection
func (c *AuthController) GitHubLogin(w http.ResponseWriter, r *http.Request) {
	state, err := service.GenerateRandomState()
	if err != nil {
		http.Error(w, "Failed to generate OAuth state", http.StatusInternalServerError)
		return
	}

	// Set state cookie (HttpOnly, SameSite=Lax, 10 min expiry)
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
	})

	http.Redirect(w, r, c.authService.GitHubAuthURL(state), http.StatusTemporaryRedirect)
}

// GitHubCallback handles the OAuth callback from GitHub with state verification
func (c *AuthController) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	// 1. Verify CSRF state parameter
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value == "" {
		http.Error(w, "Missing or expired OAuth state cookie", http.StatusBadRequest)
		return
	}

	queryState := r.URL.Query().Get("state")
	if queryState == "" {
		http.Error(w, "Missing OAuth state parameter", http.StatusBadRequest)
		return
	}

	// Clear the state cookie immediately
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
	})

	// Constant-time compare to prevent timing side-channel attacks
	if subtle.ConstantTimeCompare([]byte(stateCookie.Value), []byte(queryState)) != 1 {
		http.Error(w, "Invalid OAuth state parameter", http.StatusBadRequest)
		return
	}

	// 2. Get the code from query parameters
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
	frontendURL := c.authService.GetFrontendURL() + "/auth/callback?token=" + authResponse.Token
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
