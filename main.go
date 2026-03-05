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
	"github.com/jackc/pgx/v5"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := pgx.Connect(context.Background(), cfg.GetDatabaseURL())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close(context.Background())

	// Test database connection
	if err := db.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to database successfully")

	// Initialize services
	githubService := service.NewGithubService(&cfg)
	authService := service.NewAuthService(&cfg, githubService, db)
	jobService := service.NewJobService(db)

	// Initialize controllers
	authController := controller.NewAuthController(authService)
	jobController := controller.NewJobController(jobService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Initialize router
	r := router.NewRouter(authController, jobController, authMiddleware)
	handler := r

	// Start server
	serverAddr := ":" + cfg.Port
	log.Printf("Server starting on %s", serverAddr)
	if err := http.ListenAndServe(serverAddr, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
