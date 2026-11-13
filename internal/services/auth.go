package services

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"time"
	"uuid"

	"top1/internal/config"
	"top1/internal/models"
	"top1/internal/repo"
	"top1/internal/services/mail"
	"top1/internal/tasks"
	"top1/internal/utils"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	authRepo    *repo.AuthRepository
	taskClient  *tasks.Client
	utilService *utils.UtilService
	cfg         *config.Config
}

func NewAuthService(authRepo *repo.AuthRepository, taskClient *tasks.Client, utilService *utils.UtilService, cfg *config.Config) *AuthService {
	return &AuthService{authRepo, taskClient, utilService, cfg}
}

func (s *AuthService) Register(
	ctx context.Context,
	user *models.User,
	password string,
) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Hash password.
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}

	user.PasswordHash = string(hash)
	user.Active = false
	user.Verified = false

	// Generate verification credentials.
	otp, err := s.utilService.GenerateOTP()
	if err != nil {
		return nil, err
	}

	token, err := s.utilService.GenerateVerificationToken()
	if err != nil {
		return nil, err
	}

	// Store only hashes.
	otpHash, err := s.utilService.HashOTP(otp)
	verification := &models.EmailVerification{
		OTPHash:   otpHash,
		TokenHash: s.utilService.HashToken(token),
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	// Create user and verification record transactionally.
	createdUser, err := s.authRepo.CreateWithVerification(
		ctx,
		user,
		verification,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailAlreadyExists
		}

		// fallback: in case the typed error got lost somewhere in wrapping
		if strings.Contains(err.Error(), "users_email_key") {
			return nil, ErrEmailAlreadyExists
		}

		return nil, err
	}

	// Build verification URL.
	verificationURL := strings.TrimRight(s.cfg.AppUrl, "/") +
		"/verify-email?token=" + url.QueryEscape(token)

	// Render email.
	body, err := mail.RenderVerificationEmail(
		mail.VerificationEmailData{
			FirstName:       createdUser.FirstName,
			OTP:             otp,
			VerificationURL: verificationURL,
		},
	)
	if err != nil {
		slog.Error("Failed to render verification email", "error", err)
		return createdUser, nil
	}

	// Enqueue email.
	taskInfo, err := s.taskClient.SendEmail(
		ctx,
		"email-verification:"+createdUser.ID.String(),
		createdUser.Email,
		"Verify your Top1 account",
		body,
		true,
	)
	if err != nil {
		slog.Error("Failed to enqueue verification email",
			"email", createdUser.Email,
			"error", err,
		)
		return createdUser, nil
	}

	slog.Info("Verification email queued",
		"task_id", taskInfo.ID,
		"state", taskInfo.State,
	)

	return createdUser, nil
}

func (s *AuthService) ResendVerification(
	ctx context.Context,
	email string,
) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	email = strings.ToLower(strings.TrimSpace(email))

	// Find user.
	user, err := s.authRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	// Check account state.
	if user.Verified {
		return ErrAlreadyVerified
	}

	if user.Active {
		return ErrVerificationNotAllowed
	}

	// Generate new verification credentials.
	otp, err := s.utilService.GenerateOTP()
	if err != nil {
		return err
	}

	token, err := s.utilService.GenerateVerificationToken()
	if err != nil {
		return err
	}

	// Hash credentials.
	otpHash, err := s.utilService.HashOTP(otp)
	if err != nil {
		return err
	}

	tokenHash := s.utilService.HashToken(token)
	expiresAt := time.Now().Add(30 * time.Minute)

	// Update existing verification record.
	err = s.authRepo.UpdateEmailVerification(
		ctx,
		user.ID.String(),
		otpHash,
		tokenHash,
		expiresAt,
	)
	if err != nil {
		return err
	}

	// Build verification URL.
	verificationURL := strings.TrimRight(s.cfg.AppUrl, "/") +
		"/verify-email?token=" + url.QueryEscape(token)

	// Render email.
	body, err := mail.RenderVerificationEmail(
		mail.VerificationEmailData{
			FirstName:       user.FirstName,
			OTP:             otp,
			VerificationURL: verificationURL,
		},
	)
	if err != nil {
		return err
	}

	// Enqueue email.
	taskInfo, err := s.taskClient.SendEmail(
		ctx,
		"resend-verification:"+uuid.New().String(),
		user.Email,
		"Verify your Top1 account",
		body,
		true,
	)
	if err != nil {
		slog.Error(
			"Failed to enqueue verification email",
			"email", user.Email,
			"error", err,
		)
		return err
	}

	slog.Info(
		"Verification email queued",
		"task_id", taskInfo.ID,
		"state", taskInfo.State,
	)

	return nil
}

func (s *AuthService) VerifyOTP(
	ctx context.Context,
	email string,
	code string,
) (*models.LoggedInResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	email = strings.ToLower(strings.TrimSpace(email))
	code = strings.TrimSpace(code)

	user, err := s.authRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if user.Verified {
		return nil, ErrAlreadyVerified
	}

	otpHash, err := s.utilService.HashOTP(code)
	if err != nil {
		return nil, err
	}

	err = s.authRepo.VerifyEmail(ctx, user.ID.String(), otpHash, "")
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidOTP
		}
		return nil, err
	}

	// Token generation and refresh-token persistence go here.
	accessToken, err := s.utilService.GenerateAccessToken(user.ID.String(), utils.AccessTokenDuration)
	if err != nil {
		return nil, ErrSystemFault
	}
	refreshToken, err := s.utilService.GenerateRefreshToken(user.ID.String(), utils.RefreshTokenDuration, false)
	if err != nil {
		return nil, ErrSystemFault
	}
	// Hash refresh token before storing it.
	tokenHash := s.utilService.HashToken(refreshToken)

	// Calculate expiration.
	expiresAt := time.Now().Add(utils.RefreshTokenDuration)

	// Persist refresh token.
	if err := s.authRepo.CreateRefreshToken(
		ctx,
		user.ID.String(),
		tokenHash,
		expiresAt,
	); err != nil {
		return nil, err
	}
	resp := models.LoggedInResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	return &resp, nil
}

func (s *AuthService) VerifyToken(
	ctx context.Context,
	email string,
	token string,
) (*models.LoggedInResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.authRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if user.Verified {
		return nil, ErrAlreadyVerified
	}

	tokenHash := s.utilService.HashToken(token)

	err = s.authRepo.VerifyEmail(ctx, user.ID.String(), "", tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidOTP
		}
		return nil, err
	}

	// Token generation and refresh-token persistence go here.
	accessToken, err := s.utilService.GenerateAccessToken(user.ID.String(), utils.AccessTokenDuration)
	if err != nil {
		return nil, ErrSystemFault
	}
	refreshToken, err := s.utilService.GenerateRefreshToken(user.ID.String(), utils.RefreshTokenDuration, false)
	if err != nil {
		return nil, ErrSystemFault
	}

	// Hash refresh token before storing it.
	newtokenHash := s.utilService.HashToken(refreshToken)

	// Calculate expiration.
	expiresAt := time.Now().Add(utils.RefreshTokenDuration)

	// Persist refresh token.
	if err := s.authRepo.CreateRefreshToken(
		ctx,
		user.ID.String(),
		newtokenHash,
		expiresAt,
	); err != nil {
		return nil, err
	}

	resp := models.LoggedInResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	return &resp, nil
}

func (s *AuthService) Login(
	ctx context.Context,
	email string,
	password string,
	rememberMe bool,
) (*models.LoggedInResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	email = strings.ToLower(strings.TrimSpace(email))

	// Find user.
	user, err := s.authRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}

		return nil, err
	}

	// Verify password.
	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Check account status.
	if !user.Verified {
		return nil, ErrEmailNotVerified
	}

	if !user.Active {
		return nil, ErrAccountDisabled
	}

	// Determine refresh token duration.
	refreshDuration := utils.RefreshTokenDuration
	if rememberMe {
		refreshDuration = utils.RememberMeDuration
	}

	// Generate access token.
	accessToken, err := s.utilService.GenerateAccessToken(
		user.ID.String(),
		utils.AccessTokenDuration,
	)
	if err != nil {
		return nil, err
	}

	// Generate refresh token.
	refreshToken, err := s.utilService.GenerateRefreshToken(
		user.ID.String(),
		refreshDuration,
		rememberMe,
	)
	if err != nil {
		return nil, err
	}

	// Hash refresh token before storing it.
	tokenHash := s.utilService.HashToken(refreshToken)

	// Calculate expiration.
	expiresAt := time.Now().Add(refreshDuration)

	// Persist refresh token.
	if err := s.authRepo.CreateRefreshToken(
		ctx,
		user.ID.String(),
		tokenHash,
		expiresAt,
	); err != nil {
		return nil, err
	}

	return &models.LoggedInResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) Refresh(
	ctx context.Context,
	refreshToken string,
) (*models.LoggedInResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Validate the refresh JWT.
	claims, err := s.utilService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Determine the replacement token duration.
	refreshDuration := utils.RefreshTokenDuration
	if claims.RememberMe {
		refreshDuration = utils.RememberMeDuration
	}

	// Generate replacement tokens.
	accessToken, err := s.utilService.GenerateAccessToken(
		claims.UserID,
		utils.AccessTokenDuration,
	)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := s.utilService.GenerateRefreshToken(
		claims.UserID,
		refreshDuration,
		claims.RememberMe,
	)
	if err != nil {
		return nil, err
	}

	// Hash old and new refresh tokens.
	oldTokenHash := s.utilService.HashToken(refreshToken)
	newTokenHash := s.utilService.HashToken(newRefreshToken)

	// Atomically revoke the old token and store the replacement.
	err = s.authRepo.RotateRefreshToken(
		ctx,
		claims.UserID,
		oldTokenHash,
		newTokenHash,
		time.Now().Add(refreshDuration),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	return &models.LoggedInResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *AuthService) Logout(
	ctx context.Context,
	refreshToken string,
) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	claims, err := s.utilService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return ErrInvalidRefreshToken
	}

	tokenHash := s.utilService.HashToken(refreshToken)

	err = s.authRepo.RevokeRefreshToken(
		ctx,
		claims.UserID,
		tokenHash,
	)
	if err != nil {
		if errors.Is(err, repo.ErrRefreshTokenNotFound) {
			return ErrInvalidRefreshToken
		}
		return err
	}

	return nil
}
