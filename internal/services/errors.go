package services

import "errors"

var ErrEmailAlreadyExists = errors.New("email already exists")
var (
	ErrUserNotFound    = errors.New("user not found")
	ErrAlreadyVerified = errors.New("email already verified")
	ErrInvalidOTP      = errors.New("invalid or expired OTP")
	ErrSystemFault     = errors.New("Internal System Error Occured")
)

var (
	ErrInvalidCredentials     = errors.New("invalid email or password")
	ErrEmailNotVerified       = errors.New("email not verified")
	ErrAccountDisabled        = errors.New("account is disabled")
	ErrInvalidRefreshToken    = errors.New("invalid refresh token")
	ErrVerificationNotAllowed = errors.New("Account Verification Not alllowed")
)
