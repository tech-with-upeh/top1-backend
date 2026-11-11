package services

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"time"

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

	err = s.authRepo.VerifyEmail(ctx, user.ID.String(), otpHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidOTP
		}
		return nil, err
	}

	// Token generation and refresh-token persistence go here.
	accessToken, err := s.utilService.GenerateAccessToken(user.ID.String(), time.Hour)
	if err != nil {
		return nil, ErrSystemFault
	}
	refrshToken, err := s.utilService.GenerateRefreshToken()
	if err != nil {
		return nil, ErrSystemFault
	}
	resp := models.LoggedInResponse{
		AccessToken:  accessToken,
		RefreshToken: refrshToken,
	}
	return &resp, nil
}
