package controller

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	db "github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type AdminController struct {
	db  *db.Queries
	ctx context.Context
}

func NewAdminController(db *db.Queries, ctx context.Context) *AdminController {
	return &AdminController{db, ctx}
}

type UpdateTalentStatusRequest struct {
	TalentStatus      string `json:"talent_status"`
	AvailabilityStatus string `json:"availability_status"`
	Notes             string `json:"notes"`
	Tags              []string `json:"tags"`
	Rating            int32  `json:"rating"`
}

func (ac *AdminController) SearchTalents(ctx *gin.Context) {
	talentStatus := ctx.Query("talent_status")
	availabilityStatus := ctx.Query("availability_status")
	experienceLevel := ctx.Query("experience_level")
	location := ctx.Query("location")
	techStackStr := ctx.Query("tech_stack")
	limit := ctx.DefaultQuery("limit", "20")
	offset := ctx.DefaultQuery("offset", "0")

	var limitInt int32 = 20
	var offsetInt int32 = 0
	fmt.Sscanf(limit, "%d", &limitInt)
	fmt.Sscanf(offset, "%d", &offsetInt)

	var techStackJSON []byte
	if techStackStr != "" {
		// Parse tech stack filter (comma-separated)
		techs := strings.Split(techStackStr, ",")
		techMap := make(map[string]string)
		for _, tech := range techs {
			tech = strings.TrimSpace(tech)
			if tech != "" {
				techMap[tech] = "expert" // Default level
			}
		}
		if len(techMap) > 0 {
			techStackJSON, _ = json.Marshal(techMap)
		}
	}

	var talentStatusStr string
	if talentStatus != "" {
		talentStatusStr = talentStatus
	}

	var availabilityStatusStr string
	if availabilityStatus != "" {
		availabilityStatusStr = availabilityStatus
	}

	var experienceLevelStr string
	if experienceLevel != "" {
		experienceLevelStr = experienceLevel
	}

	var locationStr string
	if location != "" {
		locationStr = location
	}

	var techStackPtr []byte
	if len(techStackJSON) > 0 {
		techStackPtr = techStackJSON
	}

	talents, err := ac.db.SearchTalents(ac.ctx, db.SearchTalentsParams{
		Column1: talentStatusStr,
		Column2: availabilityStatusStr,
		Column3: experienceLevelStr,
		Column4: locationStr,
		Column5: techStackPtr,
		Limit:   limitInt,
		Offset:  offsetInt,
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
		"data":   talents,
	})
}

func (ac *AdminController) GetTalentById(ctx *gin.Context) {
	talentID := ctx.Param("id")
	parsedUUID, err := uuid.Parse(talentID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid talent ID",
		})
		return
	}

	talent, err := ac.db.GetTalentById(ac.ctx, pgtype.UUID{Bytes: parsedUUID, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{
				"status":  "failed",
				"message": "Talent not found",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	// Get repositories
	repos, _ := ac.db.GetRepositoriesByUserId(ac.ctx, pgtype.UUID{Bytes: parsedUUID, Valid: true})

	// Get applications
	applications, _ := ac.db.GetTalentApplications(ac.ctx, pgtype.UUID{Bytes: parsedUUID, Valid: true})

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"talent":       talent,
			"repositories": repos,
			"applications": applications,
		},
	})
}

func (ac *AdminController) UpdateTalentStatus(ctx *gin.Context) {
	talentID := ctx.Param("id")
	parsedUUID, err := uuid.Parse(talentID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid talent ID",
		})
		return
	}

	var payload *UpdateTalentStatusRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	var talentStatusText pgtype.Text
	if payload.TalentStatus != "" {
		talentStatusText = utils.StringToText(payload.TalentStatus)
	}

	var availabilityStatusText pgtype.Text
	if payload.AvailabilityStatus != "" {
		availabilityStatusText = utils.StringToText(payload.AvailabilityStatus)
	}

	updateParams := db.UpdateTalentStatusParams{
		ID:                 pgtype.UUID{Bytes: parsedUUID, Valid: true},
		TalentStatus:       talentStatusText,
		AvailabilityStatus: availabilityStatusText,
	}

	talent, err := ac.db.UpdateTalentStatus(ac.ctx, updateParams)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   talent,
	})
}

func (ac *AdminController) GetMatchingTalentsForJob(ctx *gin.Context) {
	jobID := ctx.Param("jobId")
	parsedUUID, err := uuid.Parse(jobID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid job ID",
		})
		return
	}

	limit := ctx.DefaultQuery("limit", "10")
	offset := ctx.DefaultQuery("offset", "0")

	var limitInt int32 = 10
	var offsetInt int32 = 0
	fmt.Sscanf(limit, "%d", &limitInt)
	fmt.Sscanf(offset, "%d", &offsetInt)

	talents, err := ac.db.GetMatchingTalentsForJob(ac.ctx, db.GetMatchingTalentsForJobParams{
		ID:     pgtype.UUID{Bytes: parsedUUID, Valid: true},
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
		"data":   talents,
	})
}
