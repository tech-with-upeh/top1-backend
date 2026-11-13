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

type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email,max=254"`
}

// ResendVerification godoc
//
// @Summary      Resend verification email
// @Description  Send a new verification email to an existing, inactive, unverified account.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body ResendVerificationRequest true "Email address"
// @Success      200 {object} map[string]string "Verification email queued"
// @Failure      400 {object} map[string]string "Invalid request"
// @Failure      404 {object} map[string]string "User not found"
// @Failure      409 {object} map[string]string "Verification not allowed"
// @Failure      500 {object} map[string]string "Internal server error"
// @Router       /resend-verification [post]
func (h *AuthHandler) ResendVerification(ctx *gin.Context) {
	var req ResendVerificationRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid email address",
		})
		return
	}

	err := h.authService.ResendVerification(
		ctx.Request.Context(),
		req.Email,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrUserNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "User not found",
			})

		case errors.Is(err, services.ErrAlreadyVerified),
			errors.Is(err, services.ErrVerificationNotAllowed):
			ctx.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})

		default:
			slog.Error("Failed to resend verification email", "error", err)

			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "Something went wrong",
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Verification email queued successfully",
	})
}

type VerifyOtpRequest struct {
	Email string `json:"email" binding:"required,email,max=254"`
	Code  string `json:"code" binding:"required,min=6,max=6"`
}

// VerifyOTPCode godoc
//
// @Summary      Verify email OTP
// @Description  Verify a user's email address using the six-digit OTP sent during registration. Returns access and refresh tokens.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body VerifyOtpRequest true "Email and verification code"
// @Success      200 {object} models.LoggedInResponse "Email verified successfully"
// @Failure      400 {object} map[string]string "Invalid verification details or expired OTP"
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

type VerifyTokenRequest struct {
	Email string `json:"email" binding:"required,email,max=254"`
	Token string `json:"token" binding:"required"`
}

// VerifyTokenCode godoc
//
// @Summary      Verify email Token
// @Description  Verify a user's email address using the token sent during registration. Returns access and refresh tokens.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body VerifyTokenRequest true "Email and verification token"
// @Success      200 {object} models.LoggedInResponse "Email verified successfully"
// @Failure      400 {object} map[string]string "Invalid verification details or expired OTP"
// @Failure      500 {object} map[string]string "Internal server error"
// @Router       /verify-token [post]
func (h *AuthHandler) VerifyTokenCode(ctx *gin.Context) {
	var req VerifyTokenRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid verification details",
		})
		return
	}

	response, err := h.authService.VerifyToken(
		ctx.Request.Context(),
		req.Email,
		req.Token,
	)
	if err != nil {
		slog.Error("Failed to verify Token", "error", err)

		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid or expired verification token",
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

type LoginRequest struct {
	Email      string `json:"email" binding:"required,email,max=254"`
	Password   string `json:"password" binding:"required,min=8,max=72"`
	RememberMe bool   `json:"remember_me"`
}

// Login godoc
//
// @Summary      Login user
// @Description  Authenticate a user using email and password. Returns access and refresh JWTs.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login credentials"
// @Success      200 {object} models.LoggedInResponse "Login successful"
// @Failure      400 {object} map[string]string "Invalid request"
// @Failure      401 {object} map[string]string "Invalid email or password"
// @Failure      403 {object} map[string]string "Email not verified or account disabled"
// @Failure      500 {object} map[string]string "Internal server error"
// @Router       /login [post]
func (h *AuthHandler) Login(ctx *gin.Context) {
	var req LoginRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	response, err := h.authService.Login(
		ctx.Request.Context(),
		req.Email,
		req.Password,
		req.RememberMe,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidCredentials):
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, services.ErrEmailNotVerified):
			ctx.JSON(http.StatusForbidden, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, services.ErrAccountDisabled):
			ctx.JSON(http.StatusForbidden, gin.H{
				"error": err.Error(),
			})

		default:
			slog.Error("Login failed", "error", err)

			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "Something went wrong",
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, response)
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh godoc
//
// @Summary      Refresh authentication tokens
// @Description  Validate and rotate a refresh JWT. The old refresh token is revoked and a new access and refresh token pair is returned.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body RefreshRequest true "Refresh token"
// @Success      200 {object} models.LoggedInResponse "Tokens refreshed successfully"
// @Failure      400 {object} map[string]string "Invalid request"
// @Failure      401 {object} map[string]string "Invalid, expired or revoked refresh token"
// @Failure      500 {object} map[string]string "Internal server error"
// @Router       /refresh [post]
func (h *AuthHandler) Refresh(ctx *gin.Context) {
	var req RefreshRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	response, err := h.authService.Refresh(
		ctx.Request.Context(),
		req.RefreshToken,
	)
	if err != nil {
		if errors.Is(err, services.ErrInvalidRefreshToken) {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		slog.Error("Token refresh failed", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Something went wrong",
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Logout godoc
//
// @Summary      Logout user
// @Description  Revoke the supplied refresh token and terminate its session.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body LogoutRequest true "Refresh token"
// @Success      200 {object} map[string]string "Logged out successfully"
// @Failure      400 {object} map[string]string "Invalid request"
// @Failure      401 {object} map[string]string "Invalid or revoked refresh token"
// @Failure      500 {object} map[string]string "Internal server error"
// @Router       /logout [post]
func (h *AuthHandler) Logout(ctx *gin.Context) {
	var req LogoutRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := h.authService.Logout(
		ctx.Request.Context(),
		req.RefreshToken,
	)
	if err != nil {
		if errors.Is(err, services.ErrInvalidRefreshToken) {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		slog.Error("Logout failed", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Something went wrong",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}
