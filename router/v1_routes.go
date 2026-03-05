package router

import (
	"net/http"

	"github.com/Izone-hub/talent-backend/controller"
	"github.com/Izone-hub/talent-backend/middleware"
)

// V1Routes registers all v1 API routes with clean paths (no prefix needed).
// The prefix /api/v1 is handled by the top-level router via StripPrefix.
func V1Routes(
	authController *controller.AuthController,
	jobController *controller.JobController,
	authMiddleware *middleware.AuthMiddleware,
) http.Handler {

	mux := http.NewServeMux()

	// -----------------------------------------------------------------------
	// Auth routes
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /auth/github/login", authController.GitHubLogin)
	mux.HandleFunc("GET /auth/github/callback", authController.GitHubCallback)
	mux.HandleFunc("GET /auth/me", authMiddleware.Authenticate(authController.GetCurrentUser))

	// -----------------------------------------------------------------------
	// Job routes — Public
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /jobs", jobController.ListPublishedJobs)
	mux.HandleFunc("GET /jobs/{id}", jobController.GetPublishedJob)

	// -----------------------------------------------------------------------
	// Job routes — Protected
	// -----------------------------------------------------------------------
	mux.HandleFunc("POST /jobs", authMiddleware.Authenticate(jobController.CreateJob))
	mux.HandleFunc("GET /jobs/my", authMiddleware.Authenticate(jobController.ListMyJobs))
	mux.HandleFunc("PUT /jobs/{id}", authMiddleware.Authenticate(jobController.UpdateJob))
	mux.HandleFunc("PATCH /jobs/{id}/publish", authMiddleware.Authenticate(jobController.PublishJob))
	mux.HandleFunc("PATCH /jobs/{id}/close", authMiddleware.Authenticate(jobController.CloseJob))
	mux.HandleFunc("PATCH /jobs/{id}/archive", authMiddleware.Authenticate(jobController.ArchiveJob))

	return mux
}
