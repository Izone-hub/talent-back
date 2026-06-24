package router

import (
	"net/http"

	"github.com/Izone-hub/talent-backend/controller"
	"github.com/Izone-hub/talent-backend/middleware"
)

func V1Routes(
	authController *controller.AuthController,
	jobController *controller.JobController,
	cvController *controller.CvController,
	tagController *controller.TagController,
	questionController *controller.QuestionController,
	quizController *controller.QuizController,
	appController *controller.ApplicationController,
	authMiddleware *middleware.AuthMiddleware,
) http.Handler {

	mux := http.NewServeMux()

	// -----------------------------------------------------------------------
	// Auth routes
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /api/v1/auth/github/login", authController.GitHubLogin)
	mux.HandleFunc("GET /api/v1/auth/github/callback", authController.GitHubCallback)
	mux.HandleFunc("GET /api/v1/auth/me", authMiddleware.Authenticate(authController.GetCurrentUser))

	// -----------------------------------------------------------------------
	// Job routes
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /api/v1/jobs", jobController.ListPublishedJobs)
	mux.HandleFunc("GET /api/v1/jobs/{id}", jobController.GetPublishedJob)
	mux.HandleFunc("POST /api/v1/jobs", authMiddleware.Authenticate(jobController.CreateJob))
	mux.HandleFunc("GET /api/v1/jobs/my", authMiddleware.Authenticate(jobController.ListMyJobs))
	mux.HandleFunc("PUT /api/v1/jobs/{id}", authMiddleware.Authenticate(jobController.UpdateJob))
	mux.HandleFunc("PATCH /api/v1/jobs/{id}/publish", authMiddleware.Authenticate(jobController.PublishJob))
	mux.HandleFunc("PATCH /api/v1/jobs/{id}/close", authMiddleware.Authenticate(jobController.CloseJob))
	mux.HandleFunc("PATCH /api/v1/jobs/{id}/archive", authMiddleware.Authenticate(jobController.ArchiveJob))
	mux.HandleFunc("POST /api/v1/jobs/{id}/apply", authMiddleware.Authenticate(appController.ApplyForJob))
	mux.HandleFunc("GET /api/v1/jobs/{id}/applications", authMiddleware.Authenticate(authMiddleware.RequireAdmin(appController.GetJobApplications)))

	// -----------------------------------------------------------------------
	// Application routes
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /api/v1/applications/my", authMiddleware.Authenticate(appController.GetMyApplications))
	mux.HandleFunc("PATCH /api/v1/applications/{id}/accept", authMiddleware.Authenticate(authMiddleware.RequireAdmin(appController.AcceptApplication)))

	// -----------------------------------------------------------------------
	// CV routes — Protected
	// -----------------------------------------------------------------------
	mux.HandleFunc("POST /api/v1/cv/upload", authMiddleware.Authenticate(cvController.UploadCV))
	mux.HandleFunc("GET /api/v1/cv/current", authMiddleware.Authenticate(cvController.GetCurrentCV))
	mux.HandleFunc("GET /api/v1/cv/versions", authMiddleware.Authenticate(cvController.ListCVVersions))
	mux.HandleFunc("GET /api/v1/cv/{id}/download", authMiddleware.Authenticate(cvController.DownloadCV))
	mux.HandleFunc("DELETE /api/v1/cv/{id}", authMiddleware.Authenticate(cvController.DeleteCV))

	// -----------------------------------------------------------------------
	// Tags routes
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /api/v1/tags", tagController.ListTags)
	mux.HandleFunc("POST /api/v1/tags", authMiddleware.Authenticate(tagController.CreateTag))
	mux.HandleFunc("GET /api/v1/tags/{id}", tagController.GetTag)
	mux.HandleFunc("PUT /api/v1/tags/{id}", authMiddleware.Authenticate(tagController.UpdateTag))
	mux.HandleFunc("DELETE /api/v1/tags/{id}", authMiddleware.Authenticate(tagController.DeleteTag))
	mux.HandleFunc("POST /api/v1/tags/assign", authMiddleware.Authenticate(tagController.AssignTagToJob))
	mux.HandleFunc("POST /api/v1/tags/remove", authMiddleware.Authenticate(tagController.RemoveTagFromJob))
	mux.HandleFunc("GET /api/v1/tags/{id}/jobs", tagController.GetTagJobs)
	mux.HandleFunc("GET /api/v1/jobs/{id}/tags", tagController.GetJobTags)

	// -----------------------------------------------------------------------
	// Question routes
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /api/v1/questions", questionController.ListQuestions)
	mux.HandleFunc("GET /api/v1/questions/{id}", questionController.GetQuestion)
	mux.HandleFunc("POST /api/v1/questions", authMiddleware.Authenticate(authMiddleware.RequireAdmin(questionController.CreateQuestion)))
	mux.HandleFunc("PUT /api/v1/questions/{id}", authMiddleware.Authenticate(authMiddleware.RequireAdmin(questionController.UpdateQuestion)))
	mux.HandleFunc("DELETE /api/v1/questions/{id}", authMiddleware.Authenticate(authMiddleware.RequireAdmin(questionController.DeleteQuestion)))

	// -----------------------------------------------------------------------
	// Quiz routes
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /api/v1/quizzes", authMiddleware.Authenticate(quizController.ListQuizzes))
	mux.HandleFunc("GET /api/v1/quizzes/{id}", authMiddleware.Authenticate(quizController.GetQuiz))
	mux.HandleFunc("POST /api/v1/quizzes/{id}/start", authMiddleware.Authenticate(quizController.StartQuiz))
	mux.HandleFunc("GET /api/v1/quizzes/{id}/questions", authMiddleware.Authenticate(quizController.GetQuizQuestions))
	mux.HandleFunc("POST /api/v1/quizzes/{id}/answer", authMiddleware.Authenticate(quizController.SaveAnswer))
	mux.HandleFunc("POST /api/v1/quizzes/{id}/submit", authMiddleware.Authenticate(quizController.SubmitQuiz))

	// -----------------------------------------------------------------------
	// Admin routes
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /api/v1/admin/dashboard",
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
