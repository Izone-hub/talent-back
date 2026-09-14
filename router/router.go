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
	questionFeedbackController *controller.QuestionFeedbackController,
	quizController *controller.QuizController,
	quizResultFeedbackController *controller.QuizResultFeedbackController,
	quizAnswerFeedbackController *controller.QuizAnswerFeedbackController,
	appController *controller.ApplicationController,
	sandboxController *controller.SandboxController,
	intelligenceController *controller.IntelligenceController,
	savedJobController *controller.SavedJobController,
	adminController *controller.AdminController,
	surveyQuestionController *controller.SurveyQuestionController,
	healthController *controller.HealthController,
	authMiddleware *middleware.AuthMiddleware,
) http.Handler {

	mux := http.NewServeMux()

	// -----------------------------------------------------------------------
	// Health check endpoints (for Docker / Kubernetes / load balancer probes)
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /healthz", healthController.CheckHealth)
	mux.HandleFunc("GET /health", healthController.CheckHealth)

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

		mux.Handle(
			"GET /sandbox-test",
			http.StripPrefix("/sandbox-test", fs),
		)

		mux.Handle(
			"GET /sandbox-test/",
			http.StripPrefix("/sandbox-test", fs),
		)
	}

	// -----------------------------------------------------------------------
	// Mount v1 routes directly
	// -----------------------------------------------------------------------
	v1Mux := V1Routes(
		authController,
		jobController,
		cvController,
		tagController,
		questionController,
		questionFeedbackController,
		quizController,
		quizResultFeedbackController,
		quizAnswerFeedbackController,
		appController,
		sandboxController,
		intelligenceController,
		savedJobController,
		adminController,
		surveyQuestionController,
		healthController,
		authMiddleware,
	)

	mux.Handle("/api/v1/", v1Mux)

	return mux
}
