package router

import (
	"github.com/Izone-hub/talent-backend/controller"
	"github.com/gin-gonic/gin"
)

type AdminAuthRoutes struct {
	adminAuthController controller.AdminAuthController
}

func NewAdminAuthRoutes(adminAuthController controller.AdminAuthController) AdminAuthRoutes {
	return AdminAuthRoutes{adminAuthController}
}

func (ar *AdminAuthRoutes) AdminAuthRoute(rg *gin.RouterGroup) {
	router := rg.Group("admin")

	// Public admin login endpoint
	router.POST("/login", ar.adminAuthController.Login)
}
