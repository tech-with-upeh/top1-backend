package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"top1/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

type UtilService struct {
	cfg *config.Config
}

func NewUtilService(cfg *config.Config) *UtilService {
	return &UtilService{
		cfg: cfg,
	}
}

const (
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 24 * time.Hour
	RememberMeDuration   = 30 * 24 * time.Hour
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrEmptySecret  = errors.New("JWT secret cannot be empty")
)

type AccessClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateAccessToken creates and signs an access JWT.
func (u *UtilService) GenerateAccessToken(
	userID string,
	duration time.Duration,
) (string, error) {
	secret := u.cfg.JWTSecret

	if secret == "" {
		return "", ErrEmptySecret
	}

	now := time.Now()

	claims := AccessClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "top1",
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

// ValidateAccessToken parses and validates an access JWT.
func (u *UtilService) ValidateAccessToken(
	tokenString string,
) (*AccessClaims, error) {
	secret := u.cfg.JWTSecret

	if secret == "" {
		return nil, ErrEmptySecret
	}

	claims := &AccessClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}

			return []byte(secret), nil
		},
		jwt.WithIssuer("top1"),
		jwt.WithExpirationRequired(),
	)

	if err != nil || token == nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GenerateRefreshToken creates a cryptographically random opaque token.
func (u *UtilService) GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// HashRefreshToken hashes a refresh token before database storage.
func (u *UtilService) HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// GenerateOTP generates a cryptographically secure 6-digit code.
func (u *UtilService) GenerateOTP() (string, error) {
	bytes := make([]byte, 4)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	number := binary.BigEndian.Uint32(bytes) % 1000000

	return fmt.Sprintf("%06d", number), nil
}

// GenerateVerificationToken generates a secure email verification token.
func (u *UtilService) GenerateVerificationToken() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// HashToken hashes refresh tokens and verification tokens.
func (u *UtilService) HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// VerifyToken compares a raw token against its stored hash.
func (u *UtilService) VerifyToken(token, storedHash string) bool {
	expectedHash, err := hex.DecodeString(storedHash)
	if err != nil {
		return false
	}

	actualHash := sha256.Sum256([]byte(token))

	return subtle.ConstantTimeCompare(actualHash[:], expectedHash) == 1
}

// HashOTP hashes an OTP using a server-side secret.
func (u *UtilService) HashOTP(otp string) (string, error) {
	if u.cfg.JWTSecret == "" {
		return "", errors.New("OTP secret cannot be empty")
	}

	mac := hmac.New(sha256.New, []byte(u.cfg.JWTSecret))
	mac.Write([]byte(otp))

	return hex.EncodeToString(mac.Sum(nil)), nil
}

// VerifyOTP compares a submitted OTP against its stored hash.
func (u *UtilService) VerifyOTP(otp, storedHash string) (bool, error) {
	hash, err := u.HashOTP(otp)
	if err != nil {
		return false, err
	}

	expectedHash, err := hex.DecodeString(storedHash)
	if err != nil {
		return false, nil
	}

	actualHash, _ := hex.DecodeString(hash)

	return subtle.ConstantTimeCompare(actualHash, expectedHash) == 1, nil
}
