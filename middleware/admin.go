package middleware

import (
	"net/http"
	"strings"

	"github.com/Izone-hub/talent-backend/utils"
	"github.com/gin-gonic/gin"
)

func AdminMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"status":  "failed",
				"message": "Authorization header is required",
			})
			ctx.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"status":  "failed",
				"message": "Invalid authorization header format",
			})
			ctx.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := utils.ValidateToken(tokenString, jwtSecret)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"status":  "failed",
				"message": "Invalid or expired token",
			})
			ctx.Abort()
			return
		}

		// Check if user is admin
		if claims.Role != "admin" {
			ctx.JSON(http.StatusForbidden, gin.H{
				"status":  "failed",
				"message": "Admin access required",
			})
			ctx.Abort()
			return
		}

		// Set user info in context
		ctx.Set("user_id", claims.UserID)
		ctx.Set("email", claims.Email)
		ctx.Set("role", claims.Role)
		ctx.Next()
	}
}
