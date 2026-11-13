package handlers

import (
	"net/http"

	"top1/internal/services"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *services.UserService
}

func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetMe godoc
// @Summary      Get authenticated user
// @Description  Returns the currently authenticated user's account information.
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "User retrieved successfully"
// @Failure      401 {object} map[string]string "Unauthorized or invalid access token"
// @Failure      403 {object} map[string]string "Account unavailable"
// @Failure      500 {object} map[string]string "Internal server error"
// @Router       /users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	user, err := h.userService.GetUser(
		c.Request.Context(),
		userID.(string),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve user",
		})
		return
	}

	if !user.Active || !user.Verified {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Account is unavailable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":           user.ID,
			"first_name":   user.FirstName,
			"last_name":    user.LastName,
			"email":        user.Email,
			"phone_number": user.PhoneNumber,
			"verified":     user.Verified,
			"created_at":   user.CreatedAt,
		},
	})
}
