package router

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/Izone-hub/talent-backend/controller"
	"github.com/Izone-hub/talent-backend/middleware"
)

// NewRouter creates the top-level HTTP handler.
func NewRouter(
	authController *controller.AuthController,
	jobController *controller.JobController,
	cvController *controller.CvController,
	tagController *controller.TagController,
	questionController *controller.QuestionController,
	quizController *controller.QuizController,
	appController *controller.ApplicationController,
	sandboxController *controller.SandboxController,
	intelligenceController *controller.IntelligenceController,
	authMiddleware *middleware.AuthMiddleware,
) http.Handler {

	mux := http.NewServeMux()

	// Root path handler to check API status and avoid 404
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","message":"iZone Talent API is running"}`))
	})

	// -----------------------------------------------------------------------
	// Serve sandbox test frontend
	// -----------------------------------------------------------------------
	staticDir, _ := filepath.Abs("static")
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		fs := http.FileServer(http.Dir(staticDir))
		mux.Handle("GET /sandbox-test", http.StripPrefix("/sandbox-test", fs))
		mux.Handle("GET /sandbox-test/", http.StripPrefix("/sandbox-test", fs))
	}

	// -----------------------------------------------------------------------
	// Mount v1 routes directly (Absolute path mapping)
	// -----------------------------------------------------------------------
	v1Mux := V1Routes(authController, jobController, cvController, tagController, questionController, quizController, appController, sandboxController, intelligenceController, authMiddleware)
	mux.Handle("/api/v1/", v1Mux)

	return mux
}
