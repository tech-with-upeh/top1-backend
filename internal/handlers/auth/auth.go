package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"top1/internal/models"
	"top1/internal/services"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

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
// @Router       /register [post]
func (h *AuthHandler) Register(ctx *gin.Context) {
	var req RegisterRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid registration details",
		})
		return
	}

	user := &models.User{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		PhoneNumber: req.PhoneNumber,
	}

	createdUser, err := h.authService.Register(
		ctx.Request.Context(),
		user,
		req.Password,
	)
	if err != nil {
		if errors.Is(err, services.ErrEmailAlreadyExists) {
			ctx.JSON(http.StatusConflict, gin.H{
				"error": "An account with this email already exists",
			})
			return
		}

		slog.Error("Failed to register user", "error", err)

		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Something went wrong",
		})
		return
	}

	ctx.JSON(http.StatusCreated, createdUser)
}

type VerifyOtpRequest struct {
	Email string `json:"email" binding:"required,email,max=254"`
	Code  string `json:"code" binding:"required,min=6,max=6"`
}

// VerifyOTPCODEHandler godoc
//
// @Summary      Verify USer OTP
// @Description  Verify User 6 digit OTP CODE
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body VerifyOtpRequest true "User Verify OTP details"
// @Success      201 {object} models.LoggedInReaponse "Email Verified Successfully"
// @Failure      400 {object} map[string]string "Invalid request or email code expired"
// @Failure      500 {object} map[string]string "Internal server error"
// @Router       /verify-otp [post]
func (h *AuthHandler) VerifyOTPCode(ctx *gin.Context) {
	var req VerifyOtpRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid verification details",
		})
		return
	}

	response, err := h.authService.VerifyOTP(
		ctx.Request.Context(),
		req.Email,
		req.Code,
	)
	if err != nil {
		slog.Error("Failed to verify OTP", "error", err)

		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid or expired verification code",
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}
