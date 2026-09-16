package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Izone-hub/talent-backend/config"
	"github.com/Izone-hub/talent-backend/controller"
	"github.com/Izone-hub/talent-backend/middleware"
	"github.com/Izone-hub/talent-backend/router"
	"github.com/Izone-hub/talent-backend/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database pool
	poolConfig, err := pgxpool.ParseConfig(cfg.GetDatabaseURL())
	if err != nil {
		log.Fatalf("Failed to parse db config: %v", err)
	}
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnIdleTime = 5 * time.Minute

	db, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to database successfully")

	// Temporary migration to fix 'pending' applications
	_, err = db.Exec(
		context.Background(),
		"UPDATE job_applications SET status = 'submitted' WHERE status::text = 'pending'",
	)
	if err != nil {
		log.Printf("Failed to run status migration: %v", err)
	} else {
		log.Println("Successfully migrated any 'pending' application statuses to 'submitted'")
	}

	// Ensure quiz_attempt_questions.started_at exists for backend-authoritative question timers
	_, err = db.Exec(
		context.Background(),
		"ALTER TABLE quiz_attempt_questions ADD COLUMN IF NOT EXISTS started_at TIMESTAMP;",
	)
	if err != nil {
		log.Printf("Failed to ensure quiz_attempt_questions.started_at column exists: %v", err)
	} else {
		log.Println("Successfully ensured quiz_attempt_questions.started_at column exists")
	}

	// Ensure all tags in questions are populated in tags table
	_, err = db.Exec(context.Background(), `
		INSERT INTO tags (name, category, description, color)
		SELECT DISTINCT LOWER(TRIM(t_name)), 'skill'::tag_category, 'Auto-created tag from question array', '#6366F1'
		FROM questions, unnest(tags) AS t_name
		WHERE t_name IS NOT NULL AND TRIM(t_name) != ''
		ON CONFLICT (name) DO NOTHING
	`)
	if err != nil {
		log.Printf("Failed to sync tags from questions table: %v", err)
	} else {
		log.Println("Successfully synced question tags to global tags list")
	}

	// Ensure all questions are linked in question_tags junction table
	_, err = db.Exec(context.Background(), `
		INSERT INTO question_tags (question_id, tag_id)
		SELECT q.id, t.id
		FROM questions q
		CROSS JOIN unnest(q.tags) AS t_name
		JOIN tags t ON t.name = LOWER(TRIM(t_name))
		ON CONFLICT DO NOTHING
	`)
	if err != nil {
		log.Printf("Failed to sync question_tags junction table: %v", err)
	} else {
		log.Println("Successfully synced question_tags junction table mappings")
	}

	// Ensure company_settings table exists with a default row
	_, err = db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS company_settings (
		    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		    company_name    TEXT NOT NULL DEFAULT '',
		    company_logo    TEXT NOT NULL DEFAULT '',
		    company_website TEXT NOT NULL DEFAULT '',
		    company_location TEXT NOT NULL DEFAULT '',
		    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
		    updated_at      TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Printf("Failed to ensure company_settings table: %v", err)
	} else {
		log.Println("company_settings table ensured")
	}
	// Insert default row if none exists
	_, err = db.Exec(context.Background(), `
		INSERT INTO company_settings (company_name, company_location)
		SELECT 'iZone Hub', 'Addis Ababa, Ethiopia'
		WHERE NOT EXISTS (SELECT 1 FROM company_settings LIMIT 1)
	`)
	if err != nil {
		log.Printf("Failed to seed company_settings: %v", err)
	} else {
		log.Println("company_settings default row ensured")
	}

	// Ensure quiz_attempt_questions table exists
	_, err = db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS quiz_attempt_questions (
			id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			quiz_attempt_id UUID NOT NULL REFERENCES quiz_attempts(id) ON DELETE CASCADE,
			question_id     UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
			question_order  INTEGER NOT NULL,
			created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
			UNIQUE (quiz_attempt_id, question_order),
			UNIQUE (quiz_attempt_id, question_id)
		);
		CREATE INDEX IF NOT EXISTS idx_quiz_attempt_questions_question_id ON quiz_attempt_questions(question_id);
	`)
	if err != nil {
		log.Printf("Failed to ensure quiz_attempt_questions table: %v", err)
	} else {
		log.Println("quiz_attempt_questions table ensured")
	}

	// Ensure quiz_result_feedback and quiz_answer_feedback tables exist
	_, err = db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS quiz_result_feedback (
			id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			quiz_attempt_id UUID NOT NULL REFERENCES quiz_attempts(id) ON DELETE CASCADE,
			rating          VARCHAR(10) NOT NULL CHECK (rating IN ('positive', 'negative')),
			comment         TEXT,
			created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMP NOT NULL DEFAULT NOW(),
			UNIQUE (user_id, quiz_attempt_id)
		);
		CREATE INDEX IF NOT EXISTS idx_quiz_result_feedback_attempt_id ON quiz_result_feedback(quiz_attempt_id);

		CREATE TABLE IF NOT EXISTS quiz_answer_feedback (
			id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			quiz_attempt_id UUID NOT NULL REFERENCES quiz_attempts(id) ON DELETE CASCADE,
			question_id     UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
			application_id  UUID NOT NULL REFERENCES job_applications(id) ON DELETE CASCADE,
			feedback        VARCHAR(30) NOT NULL CHECK (feedback IN ('not_related', 'not_enough_time', 'too_difficult', 'unclear_question')),
			created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMP NOT NULL DEFAULT NOW(),
			UNIQUE (user_id, quiz_attempt_id, question_id)
		);
		CREATE INDEX IF NOT EXISTS idx_quiz_answer_feedback_attempt_id ON quiz_answer_feedback(quiz_attempt_id);
		CREATE INDEX IF NOT EXISTS idx_quiz_answer_feedback_application_id ON quiz_answer_feedback(application_id);
	`)
	if err != nil {
		log.Printf("Failed to ensure quiz feedback tables: %v", err)
	} else {
		log.Println("quiz feedback tables ensured")
	}

	// Initialize services
	githubService := service.NewGithubService(&cfg)
	authService := service.NewAuthService(&cfg, githubService, db)
	jobService := service.NewJobService(db)
	clamavScanner := service.NewClamAVScanner()
	cvService := service.NewCvService(db, clamavScanner)
	tagService := service.NewTagService(db)
	questionService := service.NewQuestionService(db)

	// Initialize Analyzer Client and start bounded background worker pool (2 workers)
	analyzerClient := service.NewAnalyzerClient(cfg.AnalyzerURL, cfg.InternalServiceToken)
	cvWorker := service.NewCVAnalysisWorker(db, analyzerClient, service.CVUploadDir, 2)
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()
	cvWorker.Start(workerCtx)
	defer cvWorker.Stop()

	// Initialize and start background retention worker (runs daily in small batches)
	retentionWorker := service.NewRetentionWorker(db, 24*time.Hour)
	retentionWorker.Start(workerCtx)
	defer retentionWorker.Stop()

	sandboxService := service.NewSandboxService()

	// Initialize controllers
	authController := controller.NewAuthController(authService)
	jobController := controller.NewJobController(jobService)
	cvController := controller.NewCvController(cvService)
	tagController := controller.NewTagController(tagService)
	questionController := controller.NewQuestionController(questionService)
	questionFeedbackController := controller.NewQuestionFeedbackController(service.NewQuestionFeedbackService(db))
	sandboxController := controller.NewSandboxController(sandboxService)
	intelligenceController := controller.NewIntelligenceController(githubService, db, cfg.AnalyzerURL, cfg.InternalServiceToken)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Initialize quiz and application services
	quizService := service.NewQuizService(db)
	appService := service.NewApplicationService(db)
	quizResultFeedbackService := service.NewQuizResultFeedbackService(db)
	quizResultFeedbackController := controller.NewQuizResultFeedbackController(quizResultFeedbackService)

	quizController := controller.NewQuizController(quizService)
	appController := controller.NewApplicationController(appService, cvService, quizResultFeedbackService, cfg.AnalyzerURL, cfg.InternalServiceToken)

	savedJobController := controller.NewSavedJobController(db)
	adminController := controller.NewAdminController(
		service.NewAdminService(db),
		cfg.AnalyzerURL,
		cfg.InternalServiceToken,
	)
	surveyQuestionService := service.NewSurveyQuestionService(db)
	surveyQuestionController := controller.NewSurveyQuestionController(surveyQuestionService, jobService)

	quizAnswerFeedbackService := service.NewQuizAnswerFeedbackService(db)
	quizAnswerFeedbackController := controller.NewQuizAnswerFeedbackController(quizAnswerFeedbackService)
	healthController := controller.NewHealthController(db)

	// Create router
	handler := router.NewRouter(
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

	// Wrap handler with logging and CORS middleware
	handler = middleware.RequestLogger(handler)
	corsHandler := middleware.CORSMiddleware(handler, cfg.CORSAllowedOrigins)

	// Start server
	serverAddr := ":" + cfg.Port
	log.Printf("Server starting on %s", serverAddr)

	if err := http.ListenAndServe(serverAddr, corsHandler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
