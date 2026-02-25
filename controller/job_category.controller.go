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

type JobCategoryController struct {
	db  *db.Queries
	ctx context.Context
}

func NewJobCategoryController(db *db.Queries, ctx context.Context) *JobCategoryController {
	return &JobCategoryController{db, ctx}
}

type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

func (jcc *JobCategoryController) CreateCategory(ctx *gin.Context) {
	var payload *CreateCategoryRequest

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	categoryID := uuid.New()
	createParams := db.CreateJobCategoryParams{
		ID:          pgtype.UUID{Bytes: categoryID, Valid: true},
		Name:        payload.Name,
		Description: utils.StringToText(payload.Description),
	}

	category, err := jcc.db.CreateJobCategory(jcc.ctx, createParams)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   category,
	})
}

func (jcc *JobCategoryController) GetCategoryById(ctx *gin.Context) {
	categoryID := ctx.Param("id")
	parsedUUID, err := uuid.Parse(categoryID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid category ID",
		})
		return
	}

	category, err := jcc.db.GetJobCategoryById(jcc.ctx, pgtype.UUID{Bytes: parsedUUID, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{
				"status":  "failed",
				"message": "Category not found",
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
		"data":   category,
	})
}

func (jcc *JobCategoryController) ListCategories(ctx *gin.Context) {
	limit := ctx.DefaultQuery("limit", "10")
	offset := ctx.DefaultQuery("offset", "0")

	var limitInt int32 = 10
	var offsetInt int32 = 0
	fmt.Sscanf(limit, "%d", &limitInt)
	fmt.Sscanf(offset, "%d", &offsetInt)

	categories, err := jcc.db.ListJobCategories(jcc.ctx, db.ListJobCategoriesParams{
		Limit:  limitInt,
		Offset: offsetInt,
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
		"data":   categories,
	})
}

func (jcc *JobCategoryController) UpdateCategory(ctx *gin.Context) {
	categoryID := ctx.Param("id")
	parsedUUID, err := uuid.Parse(categoryID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid category ID",
		})
		return
	}

	var payload *CreateCategoryRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	updateParams := db.UpdateJobCategoryParams{
		ID:          pgtype.UUID{Bytes: parsedUUID, Valid: true},
		Name:        payload.Name,
		Description: utils.StringToText(payload.Description),
	}

	category, err := jcc.db.UpdateJobCategory(jcc.ctx, updateParams)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   category,
	})
}

func (jcc *JobCategoryController) DeleteCategory(ctx *gin.Context) {
	categoryID := ctx.Param("id")
	parsedUUID, err := uuid.Parse(categoryID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid category ID",
		})
		return
	}

	err = jcc.db.DeleteJobCategory(jcc.ctx, pgtype.UUID{Bytes: parsedUUID, Valid: true})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Category deleted successfully",
	})
}
