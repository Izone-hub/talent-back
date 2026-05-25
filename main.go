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

	// Initialize services
	githubService := service.NewGithubService(&cfg)
	authService := service.NewAuthService(&cfg, githubService, db)
	jobService := service.NewJobService(db)
	clamavScanner := service.NewClamAVScanner()
	cvService := service.NewCvService(db, clamavScanner)
	tagService := service.NewTagService(db)
	questionService := service.NewQuestionService(db)

	// Initialize controllers
	authController := controller.NewAuthController(authService)
	jobController := controller.NewJobController(jobService)
	cvController := controller.NewCvController(cvService)
	tagController := controller.NewTagController(tagService)
	questionController := controller.NewQuestionController(questionService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Initialize router
	handler := router.NewRouter(authController, jobController, cvController, tagController, questionController, authMiddleware)

	// Wrap handler with logging and CORS middleware
	handler = middleware.RequestLogger(handler)
	corsHandler := middleware.CORSMiddleware(handler)

	// Start server
	serverAddr := ":" + cfg.Port
	log.Printf("Server starting on %s", serverAddr)
	if err := http.ListenAndServe(serverAddr, corsHandler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
