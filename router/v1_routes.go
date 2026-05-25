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
	cvController *controller.CvController,
	tagController *controller.TagController,
	questionController *controller.QuestionController,
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

	// -----------------------------------------------------------------------
	// CV routes — Protected (all require auth)
	// -----------------------------------------------------------------------
	mux.HandleFunc("POST /cv/upload", authMiddleware.Authenticate(cvController.UploadCV))
	mux.HandleFunc("GET /cv/current", authMiddleware.Authenticate(cvController.GetCurrentCV))
	mux.HandleFunc("GET /cv/versions", authMiddleware.Authenticate(cvController.ListCVVersions))
	mux.HandleFunc("GET /cv/{id}/download", authMiddleware.Authenticate(cvController.DownloadCV))
	mux.HandleFunc("DELETE /cv/{id}", authMiddleware.Authenticate(cvController.DeleteCV))

	// -----------------------------------------------------------------------
	// Tags routes
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /tags", tagController.ListTags)
	mux.HandleFunc("POST /tags", authMiddleware.Authenticate(tagController.CreateTag)) // potentially RequireAdmin
	mux.HandleFunc("GET /tags/{id}", tagController.GetTag)
	mux.HandleFunc("PUT /tags/{id}", authMiddleware.Authenticate(tagController.UpdateTag))
	mux.HandleFunc("DELETE /tags/{id}", authMiddleware.Authenticate(tagController.DeleteTag))
	mux.HandleFunc("POST /tags/assign", authMiddleware.Authenticate(tagController.AssignTagToJob))
	mux.HandleFunc("POST /tags/remove", authMiddleware.Authenticate(tagController.RemoveTagFromJob))
	mux.HandleFunc("GET /tags/{id}/jobs", tagController.GetTagJobs)
	mux.HandleFunc("GET /jobs/{id}/tags", tagController.GetJobTags)

	// -----------------------------------------------------------------------
	// Question routes
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /questions", questionController.ListQuestions)
	mux.HandleFunc("GET /questions/{id}", questionController.GetQuestion)
	mux.HandleFunc("POST /questions", authMiddleware.Authenticate(authMiddleware.RequireAdmin(questionController.CreateQuestion)))
	mux.HandleFunc("PUT /questions/{id}", authMiddleware.Authenticate(authMiddleware.RequireAdmin(questionController.UpdateQuestion)))
	mux.HandleFunc("DELETE /questions/{id}", authMiddleware.Authenticate(authMiddleware.RequireAdmin(questionController.DeleteQuestion)))

	// -----------------------------------------------------------------------
	// Admin routes
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /admin/dashboard",
		authMiddleware.Authenticate(
			authMiddleware.RequireAdmin(
				func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Admin Dashboard"))
				},
			),
		),
	)

	return mux
}
