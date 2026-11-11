package services

import "errors"

var ErrEmailAlreadyExists = errors.New("email already exists")
var (
	ErrUserNotFound    = errors.New("user not found")
	ErrAlreadyVerified = errors.New("email already verified")
	ErrInvalidOTP      = errors.New("invalid or expired OTP")
	ErrSystemFault     = errors.New("Internal System Error Occured")
)
