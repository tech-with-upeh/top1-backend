package repo

import "errors"

var (
	ErrInvalidCredentials   = errors.New("invalid email or password")
	ErrEmailNotVerified     = errors.New("email not verified")
	ErrAccountDisabled      = errors.New("account is disabled")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)
