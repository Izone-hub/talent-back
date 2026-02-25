package router

import (
	"github.com/Izone-hub/talent-backend/controller"
	"github.com/Izone-hub/talent-backend/middleware"
	"github.com/gin-gonic/gin"
)

type ApplicationRoutes struct {
	applicationController controller.ApplicationController
	jwtSecret             string
}

func NewApplicationRoutes(applicationController controller.ApplicationController, jwtSecret string) ApplicationRoutes {
	return ApplicationRoutes{applicationController, jwtSecret}
}

func (ar *ApplicationRoutes) ApplicationRoute(rg *gin.RouterGroup) {
	router := rg.Group("applications")

	// User routes (protected)
	protected := router.Group("")
	protected.Use(middleware.AuthMiddleware(ar.jwtSecret))
	protected.POST("/", ar.applicationController.CreateApplication)
	protected.GET("/my-applications", ar.applicationController.GetMyApplications)
	protected.GET("/:id", ar.applicationController.GetApplicationById)

	// Admin routes
	adminRoutes := router.Group("")
	adminRoutes.Use(middleware.AdminMiddleware(ar.jwtSecret))
	adminRoutes.GET("/", ar.applicationController.ListApplications)
	adminRoutes.PUT("/:id/status", ar.applicationController.UpdateApplicationStatus)
}
