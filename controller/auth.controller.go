package controller

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	db "github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/services"
	"github.com/Izone-hub/talent-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type AuthController struct {
	db            *db.Queries
	ctx           context.Context
	jwtSecret     string
	githubService *services.GitHubService
}

func NewAuthController(db *db.Queries, ctx context.Context, jwtSecret string, githubService *services.GitHubService) *AuthController {
	return &AuthController{
		db:            db,
		ctx:           ctx,
		jwtSecret:     jwtSecret,
		githubService: githubService,
	}
}

// GitHubOAuth initiates the OAuth flow
func (ac *AuthController) GitHubOAuth(ctx *gin.Context) {
	githubAuthURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=user:email,repo",
		ac.githubService.GetClientID(),
		ac.githubService.GetRedirectURL(),
	)

	ctx.Redirect(http.StatusTemporaryRedirect, githubAuthURL)
}

// GitHubCallback handles the OAuth callback
func (ac *AuthController) GitHubCallback(ctx *gin.Context) {
	code := ctx.Query("code")
	if code == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status": "failed",
			"error":  "Missing authorization code",
		})
		return
	}

	// Exchange code for access token
	accessToken, err := ac.githubService.GetAccessToken(code)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	// Get user info from GitHub
	githubUser, err := ac.githubService.GetUser(accessToken)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	// Check if user already exists
	existingUser, err := ac.db.GetUserByGithubID(ac.ctx, utils.Int64ToInt8(int64(githubUser.ID)))

	var userID uuid.UUID
	var userRole string

	if err == sql.ErrNoRows {
		// New developer - create account
		userID = uuid.New()
		userRole = "applicant" // Always applicant for GitHub OAuth

		// Split name
		nameParts := strings.Fields(githubUser.Name)
		firstName := githubUser.Login
		lastName := ""
		if len(nameParts) > 0 {
			firstName = nameParts[0]
			if len(nameParts) > 1 {
				lastName = strings.Join(nameParts[1:], " ")
			}
		}

		// Create user with GitHub data
		createParams := db.CreateUserWithGithubParams{
			ID:                pgtype.UUID{Bytes: userID, Valid: true},
			FirstName:         firstName,
			LastName:          lastName,
			Email:             githubUser.Email,
			GithubUsername:    utils.StringToText(githubUser.Login),
			GithubID:          utils.Int64ToInt8(int64(githubUser.ID)),
			AvatarUrl:         utils.StringToText(githubUser.AvatarURL),
			AuthProvider:      utils.StringToText("github"),
			GithubAccessToken: utils.StringToText(accessToken),
			Bio:               utils.StringToText(githubUser.Bio),
			Location:          utils.StringToText(githubUser.Location),
			Company:           utils.StringToText(githubUser.Company),
			Blog:              utils.StringToText(githubUser.Blog),
			Role:              utils.StringToText("applicant"),
			TalentStatus:      utils.StringToText("Active"),
			AvailabilityStatus: utils.StringToText("Available"),
		}

		_, err = ac.db.CreateUserWithGithub(ac.ctx, createParams)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"status": "failed",
				"error":  err.Error(),
			})
			return
		}

		// Fetch and save repositories
		repos, err := ac.githubService.GetUserRepos(accessToken)
		if err == nil {
			ac.saveRepositories(userID, repos)
			// Aggregate tech stack
			ac.updateAggregatedTechStack(userID, repos)
		}

	} else if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	} else {
		// Existing user
		userID = uuid.UUID(existingUser.ID.Bytes)
		userRole = utils.GetTextValue(existingUser.Role)

		// Check if this is an admin trying to use GitHub OAuth
		if userRole == "admin" {
			ctx.JSON(http.StatusForbidden, gin.H{
				"status":  "failed",
				"message": "Admins must use email/password login. GitHub OAuth is for developers only.",
			})
			return
		}

		// Update GitHub data
		updateParams := db.UpdateUserGithubDataParams{
			ID:                pgtype.UUID{Bytes: userID, Valid: true},
			GithubAccessToken: utils.StringToText(accessToken),
			LastActiveAt:      pgtype.Timestamp{Valid: true, Time: time.Now()},
		}
		ac.db.UpdateUserGithubData(ac.ctx, updateParams)

		// Refresh repositories
		repos, err := ac.githubService.GetUserRepos(accessToken)
		if err == nil {
			// Delete old repositories
			ac.db.DeleteRepositoriesByUserId(ac.ctx, pgtype.UUID{Bytes: userID, Valid: true})
			// Save new ones
			ac.saveRepositories(userID, repos)
			// Update tech stack
			ac.updateAggregatedTechStack(userID, repos)
		}
	}

	// Generate JWT token
	token, err := utils.GenerateToken(userID.String(), githubUser.Email, userRole, ac.jwtSecret)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	// Redirect to frontend with token
	frontendURL := ac.githubService.GetFrontendURL()
	ctx.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s/auth/callback?token=%s", frontendURL, token))
}

func (ac *AuthController) saveRepositories(userID uuid.UUID, repos []*services.GitHubRepo) {
	for _, repo := range repos {
		// Check if repository already exists
		existingRepo, err := ac.db.GetRepositoryByGithubID(ac.ctx, repo.ID)
		if err == nil && existingRepo.ID.Valid {
			// Update existing repository
			languagesJSON, _ := json.Marshal(repo.Languages)
			techStackJSON, _ := json.Marshal(repo.TechStack)
			updateParams := db.UpdateRepositoryParams{
				ID:            existingRepo.ID,
				Description:   utils.StringToText(repo.Description),
				Language:      utils.StringToText(repo.Language),
				Languages:     languagesJSON,
				ReadmeContent: utils.StringToText(repo.Readme),
				ReadmeHtml:    utils.StringToText(repo.ReadmeHTML),
				TechStack:     techStackJSON,
				Stars:         utils.IntToInt4(repo.Stars),
				Forks:         utils.IntToInt4(repo.Forks),
			}
			ac.db.UpdateRepository(ac.ctx, updateParams)
		} else {
			// Create new repository
			languagesJSON, _ := json.Marshal(repo.Languages)
			techStackJSON, _ := json.Marshal(repo.TechStack)
			repoID := uuid.New()
			createParams := db.CreateRepositoryParams{
				ID:            pgtype.UUID{Bytes: repoID, Valid: true},
				UserID:        pgtype.UUID{Bytes: userID, Valid: true},
				GithubRepoID:  repo.ID,
				Name:          repo.Name,
				FullName:       repo.FullName,
				Description:   utils.StringToText(repo.Description),
				Url:            repo.URL,
				HtmlUrl:        repo.HTMLURL,
				Language:       utils.StringToText(repo.Language),
				Languages:      languagesJSON,
				ReadmeContent:  utils.StringToText(repo.Readme),
				ReadmeHtml:     utils.StringToText(repo.ReadmeHTML),
				TechStack:      techStackJSON,
				Stars:          utils.IntToInt4(repo.Stars),
				Forks:          utils.IntToInt4(repo.Forks),
				IsPrivate:      utils.BoolToBool(repo.Private),
			}
			ac.db.CreateRepository(ac.ctx, createParams)
		}
	}
}

func (ac *AuthController) updateAggregatedTechStack(userID uuid.UUID, repos []*services.GitHubRepo) {
	aggregated := services.AggregateTechStack(repos)
	totalStars := 0
	for _, repo := range repos {
		totalStars += repo.Stars
	}

	techStackJSON, _ := json.Marshal(aggregated)
	updateParams := db.UpdateUserGithubDataParams{
		ID:           pgtype.UUID{Bytes: userID, Valid: true},
		TechStack:    techStackJSON,
		LastActiveAt: pgtype.Timestamp{Valid: true, Time: time.Now()},
	}
	ac.db.UpdateUserGithubData(ac.ctx, updateParams)
}

// GetProfile returns current user profile
func (ac *AuthController) GetProfile(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"status":  "failed",
			"message": "User not authenticated",
		})
		return
	}

	userIDStr := userID.(string)
	parsedUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid user ID",
		})
		return
	}

	user, err := ac.db.GetUserById(ac.ctx, pgtype.UUID{Bytes: parsedUUID, Valid: true})
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"status":  "failed",
			"message": "User not found",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   user,
	})
}

// GetRepositories returns user's repositories
func (ac *AuthController) GetRepositories(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"status":  "failed",
			"message": "User not authenticated",
		})
		return
	}

	userIDStr := userID.(string)
	parsedUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "Invalid user ID",
		})
		return
	}

	repos, err := ac.db.GetRepositoriesByUserId(ac.ctx, pgtype.UUID{Bytes: parsedUUID, Valid: true})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   repos,
	})
}
