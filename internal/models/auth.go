package models

import (
	"time"
	"uuid"
)

type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	FirstName    string    `json:"first_name" db:"first_name"`
	LastName     string    `json:"last_name" db:"last_name"`
	Email        string    `json:"email" db:"email"`
	PhoneNumber  *string   `json:"phone_number" db:"phone_number"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Active       bool      `json:"active" db:"active"`
	Verified     bool      `json:"verified" db:"verified"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type EmailVerification struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	OTPHash    string     `json:"-" db:"otp_hash"`
	TokenHash  string     `json:"-" db:"token_hash"`
	ExpiresAt  time.Time  `json:"expires_at" db:"expires_at"`
	VerifiedAt *time.Time `json:"verified_at,omitempty" db:"verified_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}
type LoggedInResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
