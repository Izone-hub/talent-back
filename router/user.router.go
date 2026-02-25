package router

import (
	"github.com/Izone-hub/talent-backend/controller"
	"github.com/Izone-hub/talent-backend/middleware"
	"github.com/gin-gonic/gin"
)

type UserRoutes struct {
	userController controller.UserController
	authController controller.AuthController
	jwtSecret      string
}

func NewRouteUser(userController controller.UserController, authController controller.AuthController, jwtSecret string) UserRoutes {
	return UserRoutes{userController, authController, jwtSecret}
}

func (ur *UserRoutes) UserRoute(rg *gin.RouterGroup) {
	router := rg.Group("users")

	// Public routes
	router.POST("/", ur.userController.CreateUser)
	router.POST("/login", ur.userController.Login)

	// GitHub OAuth routes (public)
	router.GET("/auth/github", ur.authController.GitHubOAuth)
	router.GET("/auth/github/callback", ur.authController.GitHubCallback)

	// Protected routes
	protected := router.Group("")
	protected.Use(middleware.AuthMiddleware(ur.jwtSecret))
	protected.GET("/profile", ur.authController.GetProfile)
	protected.GET("/repositories", ur.authController.GetRepositories)
	protected.GET("/:userId", ur.userController.GetUserById)
}
