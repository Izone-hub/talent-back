package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Izone-hub/talent-backend/config"
	"github.com/Izone-hub/talent-backend/controller"
	dbConn "github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/router"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	_ "github.com/lib/pq"
)

var (
	server *gin.Engine
	db     *dbConn.Queries
	ctx    context.Context
	cfg    config.Config

	UserController controller.UserController
	UserRoutes     router.UserRoutes
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

	UserController = *controller.NewUserController(db, ctx, cfg.JWTSecret)
	UserRoutes = router.NewRouteUser(UserController, cfg.JWTSecret)

	server = gin.Default()
}

func main() {

	router := server.Group("/api")

	UserRoutes.UserRoute(router)

	server.NoRoute(func(ctx *gin.Context) {
		ctx.JSON(http.StatusNotFound, gin.H{"status": "failed", "message": fmt.Sprintf("The specified route %s not found", ctx.Request.URL)})
	})

	log.Fatal(server.Run(":" + "5000"))
}

func runMigrations(ctx context.Context, conn *pgx.Conn) error {
	schemaPath := filepath.Join("sql", "schema.sql")
	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	_, err = conn.Exec(ctx, string(schemaSQL))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	fmt.Println("Database migrations completed successfully...")
	return nil
}
