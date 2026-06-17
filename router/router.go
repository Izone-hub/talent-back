package router

import (
	"net/http"

	"github.com/Izone-hub/talent-backend/controller"
	"github.com/Izone-hub/talent-backend/middleware"
)

// NewRouter creates the top-level HTTP handler.
// It mounts versioned route groups under their respective prefixes.
func NewRouter(
	authController *controller.AuthController,
	jobController *controller.JobController,
	cvController *controller.CvController,
	tagController *controller.TagController,
	questionController *controller.QuestionController,
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

	// Mount v1 routes under /api/v1/
	mux.Handle(
		"/api/v1/",
		http.StripPrefix("/api/v1", V1Routes(authController, jobController, cvController, tagController, questionController, authMiddleware)),
	)

	return mux
}
