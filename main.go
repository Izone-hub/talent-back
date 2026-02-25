package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"strings"

	"github.com/Izone-hub/talent-backend/config"
	"github.com/Izone-hub/talent-backend/controller"
	dbConn "github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/router"
	"github.com/Izone-hub/talent-backend/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	_ "github.com/lib/pq"
)

var (
	server *gin.Engine
	db     *dbConn.Queries
	ctx    context.Context
	cfg    config.Config

	// Controllers
	UserController        controller.UserController
	AdminAuthController   controller.AdminAuthController
	AuthController        controller.AuthController
	JobController         controller.JobController
	JobCategoryController controller.JobCategoryController
	ApplicationController controller.ApplicationController
	AdminController       controller.AdminController

	// Routes
	UserRoutes        router.UserRoutes
	AdminAuthRoutes   router.AdminAuthRoutes
	JobRoutes         router.JobRoutes
	JobCategoryRoutes router.JobCategoryRoutes
	ApplicationRoutes router.ApplicationRoutes
	AdminRoutes       router.AdminRoutes

	// Services
	GitHubService *services.GitHubService
)

func init() {
	ctx = context.TODO()
	config, err := config.LoadConfig(".")

	if err != nil {
		log.Fatalf("could not loadconfig: %v", err)
	}

	cfg = config

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required. Please set it in your .env file or as an environment variable.\nExample: JWT_SECRET=your-secret-key-here")
	}

	if cfg.GitHubClientID == "" || cfg.GitHubClientSecret == "" {
		log.Fatal("GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET are required for GitHub OAuth")
	}

	conn, err := pgx.Connect(ctx, fmt.Sprintf("postgres://%s:%s@%s:%s/%s", config.PostgresUser, config.PostgresPassword, config.PostgresHost, config.PostgresPort, config.PostgresDb))
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}

	// Run database migrations
	if err := runMigrations(ctx, conn); err != nil {
		log.Fatalf("Could not run migrations: %v", err)
	}

	db = dbConn.New(conn)

	fmt.Println("PostgreSql connected successfully...")

	// Initialize GitHub Service
	GitHubService = services.NewGitHubService(
		cfg.GitHubClientID,
		cfg.GitHubClientSecret,
		cfg.GitHubRedirectURL,
		cfg.FrontendURL,
	)

	// Initialize Controllers
	UserController = *controller.NewUserController(db, ctx, cfg.JWTSecret)
	AdminAuthController = *controller.NewAdminAuthController(db, ctx, cfg.JWTSecret)
	AuthController = *controller.NewAuthController(db, ctx, cfg.JWTSecret, GitHubService)
	JobController = *controller.NewJobController(db, ctx)
	JobCategoryController = *controller.NewJobCategoryController(db, ctx)
	ApplicationController = *controller.NewApplicationController(db, ctx)
	AdminController = *controller.NewAdminController(db, ctx)

	// Initialize Routes
	UserRoutes = router.NewRouteUser(UserController, AuthController, cfg.JWTSecret)
	AdminAuthRoutes = router.NewAdminAuthRoutes(AdminAuthController)
	JobRoutes = router.NewJobRoutes(JobController, cfg.JWTSecret)
	JobCategoryRoutes = router.NewJobCategoryRoutes(JobCategoryController, cfg.JWTSecret)
	ApplicationRoutes = router.NewApplicationRoutes(ApplicationController, cfg.JWTSecret)
	AdminRoutes = router.NewAdminRoutes(AdminController, cfg.JWTSecret)

	server = gin.Default()
}

func main() {

	apiRouter := server.Group("/api")

	// Register all routes
	UserRoutes.UserRoute(apiRouter)
	AdminAuthRoutes.AdminAuthRoute(apiRouter)
	JobRoutes.JobRoute(apiRouter)
	JobCategoryRoutes.CategoryRoute(apiRouter)
	ApplicationRoutes.ApplicationRoute(apiRouter)
	AdminRoutes.AdminRoute(apiRouter)

	server.NoRoute(func(ctx *gin.Context) {
		ctx.JSON(http.StatusNotFound, gin.H{"status": "failed", "message": fmt.Sprintf("The specified route %s not found", ctx.Request.URL)})
	})

	fmt.Println("Server starting on port 5000...")
	log.Fatal(server.Run(":" + "5000"))
}

func runMigrations(ctx context.Context, conn *pgx.Conn) error {
	schemaPath := filepath.Join("sql", "schema.sql")
	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Execute schema - ignore errors for types that already exist
	_, err = conn.Exec(ctx, string(schemaSQL))
	if err != nil {
		// Check if error is about existing types - these are safe to ignore
		errStr := err.Error()
		if strings.Contains(errStr, "already exists") &&
			(strings.Contains(errStr, "type") || strings.Contains(errStr, "enum")) {
			fmt.Println("Note: Some database types already exist, continuing...")
		} else {
			return fmt.Errorf("failed to execute schema: %w", err)
		}
	}

	fmt.Println("Database migrations completed successfully...")
	return nil
}
