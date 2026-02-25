package controller

import (
	"context"
	"database/sql"
	"net/http"

	db "github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AdminAuthController struct {
	db        *db.Queries
	ctx       context.Context
	jwtSecret string
}

func NewAdminAuthController(db *db.Queries, ctx context.Context, jwtSecret string) *AdminAuthController {
	return &AdminAuthController{db, ctx, jwtSecret}
}

type AdminLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AdminLoginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	} `json:"user"`
}

func (ac *AdminAuthController) Login(ctx *gin.Context) {
	var payload *AdminLoginRequest

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	// Get user by email
	user, err := ac.db.GetUserByEmail(ac.ctx, payload.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"status":  "failed",
				"message": "Invalid email or password",
			})
			return
		}
		ctx.JSON(http.StatusBadGateway, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	// Check if user is admin
	userRole := utils.GetTextValue(user.Role)
	if userRole != "admin" {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"status":  "failed",
			"message": "Admin access only. Please use GitHub OAuth for developer login.",
		})
		return
	}

	// Check if user has password (admin must have password)
	passwordValue := utils.GetTextValue(user.Password)
	if passwordValue == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"status":  "failed",
			"message": "Invalid authentication method for admin",
		})
		return
	}

	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(passwordValue), []byte(payload.Password))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"status":  "failed",
			"message": "Invalid email or password",
		})
		return
	}

	// Convert UUID to string
	var userIDStr string
	if user.ID.Valid {
		userUUID := uuid.UUID(user.ID.Bytes)
		userIDStr = userUUID.String()
	} else {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  "Invalid user ID",
		})
		return
	}

	// Generate JWT token with role
	token, err := utils.GenerateToken(userIDStr, user.Email, userRole, ac.jwtSecret)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	response := AdminLoginResponse{
		Token: token,
		User: struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Role  string `json:"role"`
		}{
			ID:    userIDStr,
			Email: user.Email,
			Role:  userRole,
		},
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   response,
	})
}
