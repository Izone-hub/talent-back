package router

import (
	"github.com/Izone-hub/talent-backend/controller"
	"github.com/Izone-hub/talent-backend/middleware"
	"github.com/gin-gonic/gin"
)

type UserRoutes struct {
	userController controller.UserController
	jwtSecret      string
}

func NewRouteUser(userController controller.UserController, jwtSecret string) UserRoutes {
	return UserRoutes{userController, jwtSecret}
}

func (ur *UserRoutes) UserRoute(rg *gin.RouterGroup) {

	router := rg.Group("users")
	// Public routes
	router.POST("/", ur.userController.CreateUser)
	router.POST("/login", ur.userController.Login)

	// Protected routes
	protected := router.Group("")
	protected.Use(middleware.AuthMiddleware(ur.jwtSecret))
	protected.GET("/:userId", ur.userController.GetUserById)
}
