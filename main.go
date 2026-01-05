package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

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

	UserController controller.UserController
	UserRoutes     router.UserRoutes
)

func init() {
	ctx = context.TODO()
	config, err := config.LoadConfig(".")

	if err != nil {
		log.Fatalf("could not loadconfig: %v", err)
	}

	conn, err := pgx.Connect(ctx, fmt.Sprintf("postgres://%s:%s@%s:%s/%s", config.PostgresUser, config.PostgresPassword, config.PostgresHost, config.PostgresPort, config.PostgresDb))
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}

	db = dbConn.New(conn)

	fmt.Println("PostgreSql connected successfully...")

	UserController = *controller.NewUserController(db, ctx)
	UserRoutes = router.NewRouteUser(UserController)

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
