package controller

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	db "github.com/Izone-hub/talent-backend/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ApplicationController struct {
	db  *db.Queries
	ctx context.Context
}

func NewApplicationController(db *db.Queries, ctx context.Context) *ApplicationController {
	return &ApplicationController{db, ctx}
}

type CreateApplicationRequest struct {
	JobID string `json:"job_id" binding:"required"`
}

type UpdateApplicationStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (ac *ApplicationController) CreateApplication(ctx *gin.Context) {
	var payload *CreateApplicationRequest

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	// Get user ID from context
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"status":  "failed",
			"message": "User not authenticated",
		})
		return
	}

	userIDStr := userID.(string)
	applicantUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid user ID",
		})
		return
	}

	jobUUID, err := uuid.Parse(payload.JobID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid job ID",
		})
		return
	}

	// Check if application already exists
	count, err := ac.db.CheckApplicationExists(ac.ctx, db.CheckApplicationExistsParams{
		ApplicantID: pgtype.UUID{Bytes: applicantUUID, Valid: true},
		JobID:       pgtype.UUID{Bytes: jobUUID, Valid: true},
	})
	if err == nil && count > 0 {
		ctx.JSON(http.StatusConflict, gin.H{
			"status":  "failed",
			"message": "You have already applied to this job",
		})
		return
	}

	applicationID := uuid.New()
	createParams := db.CreateApplicationParams{
		ID:          pgtype.UUID{Bytes: applicationID, Valid: true},
		ApplicantID: pgtype.UUID{Bytes: applicantUUID, Valid: true},
		JobID:       pgtype.UUID{Bytes: jobUUID, Valid: true},
		Status:      db.ApplicationStatusPending,
	}

	application, err := ac.db.CreateApplication(ac.ctx, createParams)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   application,
	})
}

func (ac *ApplicationController) GetApplicationById(ctx *gin.Context) {
	applicationID := ctx.Param("id")
	parsedUUID, err := uuid.Parse(applicationID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid application ID",
		})
		return
	}

	application, err := ac.db.GetApplicationById(ac.ctx, pgtype.UUID{Bytes: parsedUUID, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{
				"status":  "failed",
				"message": "Application not found",
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
		"data":   application,
	})
}

func (ac *ApplicationController) ListApplications(ctx *gin.Context) {
	jobID := ctx.Query("job_id")
	status := ctx.Query("status")
	limit := ctx.DefaultQuery("limit", "10")
	offset := ctx.DefaultQuery("offset", "0")

	var limitInt int32 = 10
	var offsetInt int32 = 0
	fmt.Sscanf(limit, "%d", &limitInt)
	fmt.Sscanf(offset, "%d", &offsetInt)

	var applications interface{}
	var err error

	if jobID != "" {
		jobUUID, err := uuid.Parse(jobID)
		if err == nil {
			applications, err = ac.db.ListApplicationsByJob(ac.ctx, db.ListApplicationsByJobParams{
				JobID:  pgtype.UUID{Bytes: jobUUID, Valid: true},
				Limit:  limitInt,
				Offset: offsetInt,
			})
		}
	} else if status != "" {
		var appStatus db.ApplicationStatus
		switch status {
		case "Pending":
			appStatus = db.ApplicationStatusPending
		case "QuizGenerated":
			appStatus = db.ApplicationStatusQuizGenerated
		case "Reviewed":
			appStatus = db.ApplicationStatusReviewed
		case "Shortlisted":
			appStatus = db.ApplicationStatusShortlisted
		case "Accepted":
			appStatus = db.ApplicationStatusAccepted
		case "Rejected":
			appStatus = db.ApplicationStatusRejected
		case "InTalentPool":
			appStatus = db.ApplicationStatusInTalentPool
		default:
			appStatus = db.ApplicationStatusPending
		}
		applications, err = ac.db.ListApplicationsByStatus(ac.ctx, db.ListApplicationsByStatusParams{
			Status: appStatus,
			Limit:  limitInt,
			Offset: offsetInt,
		})
	} else {
		applications, err = ac.db.ListApplications(ac.ctx, db.ListApplicationsParams{
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
		"data":   applications,
	})
}

func (ac *ApplicationController) GetMyApplications(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"status":  "failed",
			"message": "User not authenticated",
		})
		return
	}

	userIDStr := userID.(string)
	applicantUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid user ID",
		})
		return
	}

	limit := ctx.DefaultQuery("limit", "10")
	offset := ctx.DefaultQuery("offset", "0")

	var limitInt int32 = 10
	var offsetInt int32 = 0
	fmt.Sscanf(limit, "%d", &limitInt)
	fmt.Sscanf(offset, "%d", &offsetInt)

	applications, err := ac.db.ListApplicationsByApplicant(ac.ctx, db.ListApplicationsByApplicantParams{
		ApplicantID: pgtype.UUID{Bytes: applicantUUID, Valid: true},
		Limit:       limitInt,
		Offset:      offsetInt,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   applications,
	})
}

func (ac *ApplicationController) UpdateApplicationStatus(ctx *gin.Context) {
	applicationID := ctx.Param("id")
	parsedUUID, err := uuid.Parse(applicationID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid application ID",
		})
		return
	}

	var payload *UpdateApplicationStatusRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	var appStatus db.ApplicationStatus
	switch payload.Status {
	case "Pending":
		appStatus = db.ApplicationStatusPending
	case "QuizGenerated":
		appStatus = db.ApplicationStatusQuizGenerated
	case "Reviewed":
		appStatus = db.ApplicationStatusReviewed
	case "Shortlisted":
		appStatus = db.ApplicationStatusShortlisted
	case "Accepted":
		appStatus = db.ApplicationStatusAccepted
	case "Rejected":
		appStatus = db.ApplicationStatusRejected
	case "InTalentPool":
		appStatus = db.ApplicationStatusInTalentPool
	default:
		appStatus = db.ApplicationStatusPending
	}

	updateParams := db.UpdateApplicationStatusParams{
		ID:     pgtype.UUID{Bytes: parsedUUID, Valid: true},
		Status: appStatus,
	}

	application, err := ac.db.UpdateApplicationStatus(ac.ctx, updateParams)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   application,
	})
}
