package controller

import (
	"context"
	"database/sql"
	"net/http"

	db "github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type UserController struct {
	db        *db.Queries
	ctx       context.Context
	jwtSecret string
}

type CreateUser struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required"`
}

func NewUserController(db *db.Queries, ctx context.Context, jwtSecret string) *UserController {
	return &UserController{db, ctx, jwtSecret}
}

func (uc *UserController) CreateUser(ctx *gin.Context) {
	var payload *CreateUser

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "Failed payload", "error": err.Error()})
		return
	}

	// Hash password with bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": "Failed to hash password", "error": err.Error()})
		return
	}

	newUUID := uuid.New()
	args := &db.CreateUserParams{
		ID:        pgtype.UUID{Bytes: newUUID, Valid: true},
		FirstName: payload.FirstName,
		LastName:  payload.LastName,
		Email:     payload.Email,
		Password:  utils.StringToText(string(hashedPassword)),
	}

	user, err := uc.db.CreateUser(ctx, *args)

	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"status": "Failed retrieving user", "error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "successfully created user", "user": user})
}

type GetUserById struct {
	ID string `json:"id" binding:"required"`
}

func (uc *UserController) GetUserById(ctx *gin.Context) {
	userId := ctx.Param("userId")

	parsedUUID, err := uuid.Parse(userId)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "failed", "message": "Invalid user ID format"})
		return
	}

	pgUUID := pgtype.UUID{Bytes: parsedUUID, Valid: true}
	user, err := uc.db.GetUserById(ctx, pgUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{"status": "failed", "message": "Failed to retrieve user with this ID"})
			return
		}
		ctx.JSON(http.StatusBadGateway, gin.H{"status": "Failed retrieving user", "error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "Successfully retrived id", "user": user})
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

func (uc *UserController) Login(ctx *gin.Context) {
	var payload *LoginRequest

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "Failed payload", "error": err.Error()})
		return
	}

	// Get user by email
	user, err := uc.db.GetUserByEmail(ctx, payload.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusUnauthorized, gin.H{"status": "failed", "message": "Invalid email or password"})
			return
		}
		ctx.JSON(http.StatusBadGateway, gin.H{"status": "Failed retrieving user", "error": err.Error()})
		return
	}

	// Compare password with hashed password
	passwordValue := utils.GetTextValue(user.Password)
	err = bcrypt.CompareHashAndPassword([]byte(passwordValue), []byte(payload.Password))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"status": "failed", "message": "Invalid email or password"})
		return
	}

	// Convert pgtype.UUID to string
	var userIDStr string
	var parsedUUID uuid.UUID
	if user.ID.Valid {
		parsedUUID = uuid.UUID(user.ID.Bytes)
		userIDStr = parsedUUID.String()
	} else {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": "Invalid user ID"})
		return
	}

	// Get user role from database
	userFull, err := uc.db.GetUserById(uc.ctx, pgtype.UUID{Bytes: parsedUUID, Valid: true})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": "Invalid user ID"})
		return
	}

	// Generate JWT token with role
	userRole := utils.GetTextValue(userFull.Role)
	if userRole == "" {
		userRole = "applicant" // Default role
	}
	token, err := utils.GenerateToken(userIDStr, user.Email, userRole, uc.jwtSecret)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": "Failed to generate token", "error": err.Error()})
		return
	}

	response := LoginResponse{
		Token: token,
		User: struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		}{
			ID:    userIDStr,
			Email: user.Email,
		},
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "success", "data": response})
}
