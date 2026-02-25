package router

import (
	"github.com/Izone-hub/talent-backend/controller"
	"github.com/Izone-hub/talent-backend/middleware"
	"github.com/gin-gonic/gin"
)

type AdminRoutes struct {
	adminController controller.AdminController
	jwtSecret       string
}

func NewAdminRoutes(adminController controller.AdminController, jwtSecret string) AdminRoutes {
	return AdminRoutes{adminController, jwtSecret}
}

func (ar *AdminRoutes) AdminRoute(rg *gin.RouterGroup) {
	router := rg.Group("admin")

	// All routes require admin middleware
	router.Use(middleware.AdminMiddleware(ar.jwtSecret))

	router.GET("/talents", ar.adminController.SearchTalents)
	router.GET("/talents/:id", ar.adminController.GetTalentById)
	router.PUT("/talents/:id/status", ar.adminController.UpdateTalentStatus)
	router.GET("/jobs/:jobId/matching-talents", ar.adminController.GetMatchingTalentsForJob)
}
