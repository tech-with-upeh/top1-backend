package middleware

import (
	"net/http"
	"strings"

	"top1/internal/utils"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(utilService *utils.UtilService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		parts := strings.Fields(authHeader)

		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Missing or invalid authorization header",
			})
			return
		}

		claims, err := utilService.ValidateAccessToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired access token",
			})
			return
		}

		c.Set("userID", claims.UserID)

		c.Next()
	}
}
