package auth

import (
	"net/http"
	"strings"

	"top1/internal/models"
	"top1/internal/repo"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	FirstName   string  `json:"first_name" binding:"required,min=2,max=50"`
	LastName    string  `json:"last_name" binding:"required,min=2,max=50"`
	Email       string  `json:"email" binding:"required,email,max=254"`
	PhoneNumber *string `json:"phone_number" binding:"omitempty,min=5,max=30"`
	Password    string  `json:"password" binding:"required,min=8,max=72"`
}

// CreateUserHandler godoc
//
// @Summary      Register a new user
// @Description  Create a new user account with name, email, optional phone number, and password
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "User registration details"
// @Success      201 {object} models.User "User created successfully"
// @Failure      400 {object} map[string]string "Invalid request or email already exists"
// @Failure      500 {object} map[string]string "Internal server error"
// @Router       /auth/register [post]
func CreateUserHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req RegisterRequest
		if err := ctx.BindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"msg":   "Bad Request",
				"error": err.Error(),
			})
			return
		}

		hashpass, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to Hash Password ", "error": err.Error()})
			return
		}

		user := &models.User{
			FirstName:    req.FirstName,
			LastName:     req.LastName,
			Email:        req.Email,
			PhoneNumber:  req.PhoneNumber,
			PasswordHash: string(hashpass),
		}

		createdUser, err := repo.CreateUser(pool, user)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") ||
				strings.Contains(err.Error(), "23505") {
				ctx.JSON(http.StatusConflict, gin.H{
					"msg":   "Email already exists",
					"error": err.Error(),
				})
				return
			}

			ctx.JSON(http.StatusInternalServerError, gin.H{
				"msg":   "Failed to create user",
				"error": err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusCreated, createdUser)
	}
}
