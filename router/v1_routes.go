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
	sandboxController *controller.SandboxController,
	intelligenceController *controller.IntelligenceController,
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
	mux.HandleFunc("GET /api/v1/jobs", authMiddleware.OptionalAuthenticate(jobController.ListPublishedJobs))
	mux.HandleFunc("GET /api/v1/jobs/{id}", jobController.GetPublishedJob)
	mux.HandleFunc("POST /api/v1/jobs", authMiddleware.Authenticate(jobController.CreateJob))
	mux.HandleFunc("GET /api/v1/jobs/my", authMiddleware.Authenticate(jobController.ListMyJobs))
	mux.HandleFunc("PUT /api/v1/jobs/{id}", authMiddleware.Authenticate(jobController.UpdateJob))
	mux.HandleFunc("PATCH /api/v1/jobs/{id}/publish", authMiddleware.Authenticate(jobController.PublishJob))
	mux.HandleFunc("PATCH /api/v1/jobs/{id}/close", authMiddleware.Authenticate(jobController.CloseJob))
	mux.HandleFunc("PATCH /api/v1/jobs/{id}/archive", authMiddleware.Authenticate(jobController.ArchiveJob))
	mux.HandleFunc("POST /api/v1/jobs/{id}/apply", authMiddleware.Authenticate(appController.ApplyForJob))

	// -----------------------------------------------------------------------
	// Application routes
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /api/v1/applications/my", authMiddleware.Authenticate(appController.GetMyApplications))
	mux.HandleFunc("GET /api/v1/applications/recent", authMiddleware.Authenticate(authMiddleware.RequireAdmin(appController.GetRecentApplications)))
	mux.HandleFunc("GET /api/v1/applications/status/{status}", authMiddleware.Authenticate(authMiddleware.RequireAdmin(appController.ListApplicationsByStatus)))
	mux.HandleFunc("GET /api/v1/applications/{id}", authMiddleware.Authenticate(appController.GetApplicationDetail))
	mux.HandleFunc("PATCH /api/v1/applications/{id}/review", authMiddleware.Authenticate(authMiddleware.RequireAdmin(appController.StartReview)))
	mux.HandleFunc("PATCH /api/v1/applications/{id}/shortlist", authMiddleware.Authenticate(authMiddleware.RequireAdmin(appController.ShortlistApplication)))
	mux.HandleFunc("PATCH /api/v1/applications/{id}/interview", authMiddleware.Authenticate(authMiddleware.RequireAdmin(appController.MarkInterviewed)))
	mux.HandleFunc("PATCH /api/v1/applications/{id}/accept", authMiddleware.Authenticate(authMiddleware.RequireAdmin(appController.AcceptApplication)))
	mux.HandleFunc("PATCH /api/v1/applications/{id}/reject", authMiddleware.Authenticate(authMiddleware.RequireAdmin(appController.RejectApplication)))
	mux.HandleFunc("PATCH /api/v1/applications/{id}/withdraw", authMiddleware.Authenticate(appController.WithdrawApplication))
	mux.HandleFunc("PATCH /api/v1/applications/{id}/feedback", authMiddleware.Authenticate(authMiddleware.RequireAdmin(appController.AddEmployerFeedback)))
	mux.HandleFunc("GET /api/v1/jobs/{id}/applications", authMiddleware.Authenticate(authMiddleware.RequireAdmin(appController.GetJobApplications)))
	mux.HandleFunc("GET /api/v1/jobs/{id}/applications/counts", authMiddleware.Authenticate(authMiddleware.RequireAdmin(appController.GetApplicationCountsByJob)))

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
	mux.HandleFunc("GET /api/v1/tags", authMiddleware.Authenticate(tagController.ListTags))
	mux.HandleFunc("POST /api/v1/tags", authMiddleware.Authenticate(tagController.CreateTag))
	mux.HandleFunc("GET /api/v1/tags/{id}", authMiddleware.Authenticate(tagController.GetTag))
	mux.HandleFunc("PUT /api/v1/tags/{id}", authMiddleware.Authenticate(tagController.UpdateTag))
	mux.HandleFunc("DELETE /api/v1/tags/{id}", authMiddleware.Authenticate(tagController.DeleteTag))
	mux.HandleFunc("POST /api/v1/tags/assign", authMiddleware.Authenticate(authMiddleware.RequireAdmin(tagController.AssignTagToJob)))
	mux.HandleFunc("POST /api/v1/tags/remove", authMiddleware.Authenticate(authMiddleware.RequireAdmin(tagController.RemoveTagFromJob)))
	mux.HandleFunc("GET /api/v1/tags/{id}/jobs", authMiddleware.Authenticate(tagController.GetTagJobs))
	mux.HandleFunc("GET /api/v1/jobs/{id}/tags", authMiddleware.Authenticate(tagController.GetJobTags))

	// -----------------------------------------------------------------------
	// Question routes
	// -----------------------------------------------------------------------
	// Secure GET routes
	mux.HandleFunc("GET /api/v1/questions", authMiddleware.Authenticate(authMiddleware.RequireAdmin(questionController.ListQuestions)))

	mux.HandleFunc("GET /api/v1/questions/{id}", authMiddleware.Authenticate(authMiddleware.RequireAdmin(questionController.GetQuestion)))

	mux.HandleFunc("POST /api/v1/questions", authMiddleware.Authenticate(authMiddleware.RequireAdmin(questionController.CreateQuestion)))

	mux.HandleFunc("PUT /api/v1/questions/{id}", authMiddleware.Authenticate(authMiddleware.RequireAdmin(questionController.UpdateQuestion)))

	mux.HandleFunc("DELETE /api/v1/questions/{id}", authMiddleware.Authenticate(authMiddleware.RequireAdmin(questionController.DeleteQuestion)))

	// -----------------------------------------------------------------------
	// Quiz routes
	// -----------------------------------------------------------------------
	// 1. List all quizzes for the user
	mux.HandleFunc("GET /api/v1/quizzes", authMiddleware.Authenticate(quizController.ListQuizzes))

	// 2. View details of a specific quiz attempt
	mux.HandleFunc("GET /api/v1/quizzes/{id}", authMiddleware.Authenticate(quizController.GetQuiz))

	// 3. Start the quiz (creates the attempt record)
	mux.HandleFunc("POST /api/v1/quizzes/{id}/start", authMiddleware.Authenticate(quizController.StartQuiz))

	// 4. Fetch ONLY the next question (Replaces the bulk list)
	mux.HandleFunc("GET /api/v1/quizzes/{id}/question", authMiddleware.Authenticate(quizController.GetNextQuestion))

	// 5. Submit an answer for the current question
	mux.HandleFunc("POST /api/v1/quizzes/{id}/answer", authMiddleware.Authenticate(quizController.SaveAnswer))

	// 6. Run code for coding_challenge questions
	mux.HandleFunc("POST /api/v1/quizzes/{id}/run-code", authMiddleware.Authenticate(quizController.RunCode))

	// 7. Finish and score the quiz
	mux.HandleFunc("POST /api/v1/quizzes/{id}/submit", authMiddleware.Authenticate(quizController.SubmitQuiz))
	// Example registration in main.go
	mux.HandleFunc("GET /api/v1/quizzes/{id}/next", authMiddleware.Authenticate(quizController.GetNextQuestion))

	// -----------------------------------------------------------------------
	// Sandbox / Judge routes
	// -----------------------------------------------------------------------
	mux.HandleFunc("GET /api/v1/sandbox/languages", authMiddleware.Authenticate(sandboxController.ListLanguages))
	mux.HandleFunc("POST /api/v1/sandbox/execute", authMiddleware.Authenticate(sandboxController.Execute))

	// -----------------------------------------------------------------------
	// Intelligence routes
	// -----------------------------------------------------------------------
	mux.HandleFunc("POST /api/v1/intelligence/github/{id}/fetch", authMiddleware.Authenticate(intelligenceController.FetchGitHubSnapshot))
	mux.HandleFunc("POST /api/v1/analyze-cv", intelligenceController.AnalyzeCV)

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
