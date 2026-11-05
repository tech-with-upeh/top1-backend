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
