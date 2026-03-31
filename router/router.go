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
	authMiddleware *middleware.AuthMiddleware,
) http.Handler {

	mux := http.NewServeMux()

	// Mount v1 routes under /api/v1/
	mux.Handle(
		"/api/v1/",
		http.StripPrefix("/api/v1", V1Routes(authController, jobController, cvController, tagController, authMiddleware)),
	)

	return mux
}
