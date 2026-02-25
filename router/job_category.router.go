package router

import (
	"github.com/Izone-hub/talent-backend/controller"
	"github.com/Izone-hub/talent-backend/middleware"
	"github.com/gin-gonic/gin"
)

type JobCategoryRoutes struct {
	categoryController controller.JobCategoryController
	jwtSecret          string
}

func NewJobCategoryRoutes(categoryController controller.JobCategoryController, jwtSecret string) JobCategoryRoutes {
	return JobCategoryRoutes{categoryController, jwtSecret}
}

func (jcr *JobCategoryRoutes) CategoryRoute(rg *gin.RouterGroup) {
	router := rg.Group("jobs/categories")

	// Public routes
	router.GET("/", jcr.categoryController.ListCategories)
	router.GET("/:id", jcr.categoryController.GetCategoryById)

	// Admin routes
	adminRoutes := router.Group("")
	adminRoutes.Use(middleware.AdminMiddleware(jcr.jwtSecret))
	adminRoutes.POST("/", jcr.categoryController.CreateCategory)
	adminRoutes.PUT("/:id", jcr.categoryController.UpdateCategory)
	adminRoutes.DELETE("/:id", jcr.categoryController.DeleteCategory)
}
