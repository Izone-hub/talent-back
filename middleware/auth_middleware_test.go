package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Izone-hub/talent-backend/config"
	"github.com/Izone-hub/talent-backend/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRequireAdmin_WithDatabaseRoleSync(t *testing.T) {
	cfg, err := config.LoadConfig("../")
	if err != nil {
		t.Skip("Skipping DB test: config not loaded")
	}

	db, err := pgxpool.New(context.Background(), cfg.GetDatabaseURL())
	if err != nil {
		t.Skip("Skipping DB test: cannot connect to DB")
	}
	defer db.Close()

	authService := service.NewAuthService(&cfg, nil, db)
	mw := NewAuthMiddleware(authService)

	// Fetch meka15 user ID
	var userID uuid.UUID
	err = db.QueryRow(context.Background(), "SELECT id FROM users WHERE github_username = 'meka15'").Scan(&userID)
	if err != nil {
		t.Skipf("Skipping: meka15 user not found: %v", err)
	}

	// Create a dummy handler that should only be reached by admins
	handlerCalled := false
	handler := mw.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Simulate a request with a stale JWT claim where Role is "user" instead of "admin"
	staleClaims := &service.Claims{
		UserID:         userID,
		GithubUsername: "meka15",
		Role:           "user", // Stale claim!
	}

	req := httptest.NewRequest("GET", "/api/v1/jobs/my", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user", staleClaims))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK after syncing DB role, got %d. Body: %s", w.Code, w.Body.String())
	}
	if !handlerCalled {
		t.Fatalf("expected handler to be called")
	}
	if staleClaims.Role != "admin" {
		t.Fatalf("expected claims.Role to be updated to 'admin', got %s", staleClaims.Role)
	}
}
