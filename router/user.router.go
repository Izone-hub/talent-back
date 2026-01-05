package router

import (
	"github.com/Izone-hub/talent-backend/controller"
	"github.com/gin-gonic/gin"
)

type UserRoutes struct {
	userController controller.UserController
}

func NewRouteUser(userController controller.UserController) UserRoutes {
	return UserRoutes{userController}
}

func (ur *UserRoutes) UserRoute(rg *gin.RouterGroup) {

	router := rg.Group("users")
	router.POST("/", ur.userController.CreateUser)
	router.GET("/:userId", ur.userController.GetUserById)
}
