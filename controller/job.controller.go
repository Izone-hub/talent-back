package controller

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	db "github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type JobController struct {
	db  *db.Queries
	ctx context.Context
}

func NewJobController(db *db.Queries, ctx context.Context) *JobController {
	return &JobController{db, ctx}
}

type CreateJobRequest struct {
	Title        string `json:"title" binding:"required"`
	Description  string `json:"description" binding:"required"`
	Role         string `json:"role" binding:"required"`
	CategoryID   string `json:"category_id" binding:"required"`
	Requirements string `json:"requirements"`
	Location     string `json:"location"`
	JobType      string `json:"job_type"`
	Status       string `json:"status"`
}

func (jc *JobController) CreateJob(ctx *gin.Context) {
	var payload *CreateJobRequest

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	// Get admin user ID from context
	adminID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"status":  "failed",
			"message": "User not authenticated",
		})
		return
	}

	adminIDStr := adminID.(string)
	adminUUID, err := uuid.Parse(adminIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid user ID",
		})
		return
	}

	categoryUUID, err := uuid.Parse(payload.CategoryID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid category ID",
		})
		return
	}

	var jobStatus db.JobStatus
	if payload.Status == "" {
		jobStatus = db.JobStatusOpen
	} else if payload.Status == "Open" {
		jobStatus = db.JobStatusOpen
	} else {
		jobStatus = db.JobStatusClosed
	}

	var jobRole db.JobRoles
	switch payload.Role {
	case "Frontend":
		jobRole = db.JobRolesFrontend
	case "Backend":
		jobRole = db.JobRolesBackend
	case "Fullstack":
		jobRole = db.JobRolesFullstack
	case "Ui_Ux":
		jobRole = db.JobRolesUiUx
	default:
		jobRole = db.JobRolesBackend
	}

	jobID := uuid.New()
	createParams := db.CreateJobParams{
		ID:           pgtype.UUID{Bytes: jobID, Valid: true},
		Title:        payload.Title,
		Description:  payload.Description,
		Role:         jobRole,
		CategoryID:   pgtype.UUID{Bytes: categoryUUID, Valid: true},
		Requirements: utils.StringToText(payload.Requirements),
		Location:     utils.StringToText(payload.Location),
		JobType:      utils.StringToText(payload.JobType),
		Status:       jobStatus,
		CreatedBy:    pgtype.UUID{Bytes: adminUUID, Valid: true},
	}

	job, err := jc.db.CreateJob(jc.ctx, createParams)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   job,
	})
}

func (jc *JobController) GetJobById(ctx *gin.Context) {
	jobID := ctx.Param("id")
	parsedUUID, err := uuid.Parse(jobID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid job ID",
		})
		return
	}

	job, err := jc.db.GetJobById(jc.ctx, pgtype.UUID{Bytes: parsedUUID, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{
				"status":  "failed",
				"message": "Job not found",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   job,
	})
}

func (jc *JobController) ListJobs(ctx *gin.Context) {
	limit := ctx.DefaultQuery("limit", "10")
	offset := ctx.DefaultQuery("offset", "0")
	status := ctx.Query("status")
	categoryID := ctx.Query("category_id")

	var limitInt int32 = 10
	var offsetInt int32 = 0
	fmt.Sscanf(limit, "%d", &limitInt)
	fmt.Sscanf(offset, "%d", &offsetInt)

	var jobs interface{}
	var err error

	if categoryID != "" {
		categoryUUID, err := uuid.Parse(categoryID)
		if err == nil {
			jobs, err = jc.db.ListJobsByCategory(jc.ctx, db.ListJobsByCategoryParams{
				CategoryID: pgtype.UUID{Bytes: categoryUUID, Valid: true},
				Limit:      limitInt,
				Offset:     offsetInt,
			})
		}
	} else if status != "" {
		var jobStatus db.JobStatus
		if status == "Open" {
			jobStatus = db.JobStatusOpen
		} else {
			jobStatus = db.JobStatusClosed
		}
		jobs, err = jc.db.ListJobsByStatus(jc.ctx, db.ListJobsByStatusParams{
			Status: jobStatus,
			Limit:  limitInt,
			Offset: offsetInt,
		})
	} else {
		jobs, err = jc.db.ListJobs(jc.ctx, db.ListJobsParams{
			Limit:  limitInt,
			Offset: offsetInt,
		})
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   jobs,
	})
}

func (jc *JobController) UpdateJob(ctx *gin.Context) {
	jobID := ctx.Param("id")
	parsedUUID, err := uuid.Parse(jobID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid job ID",
		})
		return
	}

	var payload *CreateJobRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	categoryUUID, err := uuid.Parse(payload.CategoryID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid category ID",
		})
		return
	}

	var jobStatus db.JobStatus
	if payload.Status == "Open" {
		jobStatus = db.JobStatusOpen
	} else {
		jobStatus = db.JobStatusClosed
	}

	var jobRole db.JobRoles
	switch payload.Role {
	case "Frontend":
		jobRole = db.JobRolesFrontend
	case "Backend":
		jobRole = db.JobRolesBackend
	case "Fullstack":
		jobRole = db.JobRolesFullstack
	case "Ui_Ux":
		jobRole = db.JobRolesUiUx
	default:
		jobRole = db.JobRolesBackend
	}

	updateParams := db.UpdateJobParams{
		ID:           pgtype.UUID{Bytes: parsedUUID, Valid: true},
		Title:        payload.Title,
		Description:  payload.Description,
		Role:         jobRole,
		CategoryID:   pgtype.UUID{Bytes: categoryUUID, Valid: true},
		Requirements: utils.StringToText(payload.Requirements),
		Location:     utils.StringToText(payload.Location),
		JobType:      utils.StringToText(payload.JobType),
		Status:       jobStatus,
	}

	job, err := jc.db.UpdateJob(jc.ctx, updateParams)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   job,
	})
}

func (jc *JobController) DeleteJob(ctx *gin.Context) {
	jobID := ctx.Param("id")
	parsedUUID, err := uuid.Parse(jobID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid job ID",
		})
		return
	}

	err = jc.db.DeleteJob(jc.ctx, pgtype.UUID{Bytes: parsedUUID, Valid: true})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Job deleted successfully",
	})
}
