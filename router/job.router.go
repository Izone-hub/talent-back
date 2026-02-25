package router

import (
	"github.com/Izone-hub/talent-backend/controller"
	"github.com/Izone-hub/talent-backend/middleware"
	"github.com/gin-gonic/gin"
)

type JobRoutes struct {
	jobController controller.JobController
	jwtSecret     string
}

func NewJobRoutes(jobController controller.JobController, jwtSecret string) JobRoutes {
	return JobRoutes{jobController, jwtSecret}
}

func (jr *JobRoutes) JobRoute(rg *gin.RouterGroup) {
	router := rg.Group("jobs")

	// Public routes
	router.GET("/", jr.jobController.ListJobs)
	router.GET("/:id", jr.jobController.GetJobById)

	// Admin routes
	adminRoutes := router.Group("")
	adminRoutes.Use(middleware.AdminMiddleware(jr.jwtSecret))
	adminRoutes.POST("/", jr.jobController.CreateJob)
	adminRoutes.PUT("/:id", jr.jobController.UpdateJob)
	adminRoutes.DELETE("/:id", jr.jobController.DeleteJob)
}
