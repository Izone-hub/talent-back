package main

import (
	"context"
	"log"
	"net/http"

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
	db, err := pgxpool.New(context.Background(), cfg.GetDatabaseURL())
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
	_, err = db.Exec(context.Background(), "UPDATE job_applications SET status = 'submitted' WHERE status = 'pending'")
	if err != nil {
		log.Printf("Failed to run status migration: %v", err)
	} else {
		log.Println("Successfully migrated any 'pending' application statuses to 'submitted'")
	}

	// Ensure all tags in questions are populated in tags table
	_, err = db.Exec(context.Background(), `
		INSERT INTO tags (name, category, description, color)
		SELECT DISTINCT LOWER(TRIM(t_name)), 'skill', 'Auto-created tag from question array', '#6366F1'
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


	// Initialize services
	githubService := service.NewGithubService(&cfg)
	authService := service.NewAuthService(&cfg, githubService, db)
	jobService := service.NewJobService(db)
	clamavScanner := service.NewClamAVScanner()
	cvService := service.NewCvService(db, clamavScanner)
	tagService := service.NewTagService(db)
	questionService := service.NewQuestionService(db)

	sandboxService := service.NewSandboxService()

	// Initialize controllers
	authController := controller.NewAuthController(authService)
	jobController := controller.NewJobController(jobService)
	cvController := controller.NewCvController(cvService)
	tagController := controller.NewTagController(tagService)
	questionController := controller.NewQuestionController(questionService)
	sandboxController := controller.NewSandboxController(sandboxService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)
	// 1. Ensure the quiz controller is initialized in main.go
	// 1. Initialize the Quiz Service with the DB pool handle
	quizService := service.NewQuizService(db)
	appService := service.NewApplicationService(db)

	quizController := controller.NewQuizController(quizService)
	appController := controller.NewApplicationController(appService)

	// 2. Update the router creation call to pass it in
	handler := router.NewRouter(
		authController,
		jobController,
		cvController,
		tagController,
		questionController,
		quizController,
		appController,
		sandboxController,
		authMiddleware,
	)

	// Wrap handler with logging and CORS middleware
	handler = middleware.RequestLogger(handler)
	corsHandler := middleware.CORSMiddleware(handler)
   
	// Start server
	serverAddr := ":5000"
	log.Printf("Server starting on %s", serverAddr)
	if err := http.ListenAndServe(serverAddr, corsHandler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
