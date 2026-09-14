package repository

import "fmt"

// OTPError represents OTP-specific errors
type OTPError struct {
	Code    string
	Message string
}

func (e *OTPError) Error() string {
	return fmt.Sprintf("OTP Error [%s]: %s", e.Code, e.Message)
}

// Common OTP errors
var (
	ErrOTPRateLimited       = &OTPError{Code: "OTP_RATE_LIMITED", Message: "Maximum OTP requests exceeded. Please try again later."}
	ErrOTPResendTooSoon     = &OTPError{Code: "OTP_RESEND_TOO_SOON", Message: "Please wait before requesting a new OTP."}
	ErrOTPInvalid           = &OTPError{Code: "OTP_INVALID", Message: "Invalid or expired OTP."}
	ErrOTPVerificationFailed = &OTPError{Code: "OTP_VERIFICATION_FAILED", Message: "OTP verification failed. Please check and try again."}
	ErrOTPMaxAttemptsExceeded = &OTPError{Code: "OTP_MAX_ATTEMPTS", Message: "Maximum OTP verification attempts exceeded."}
)

// NewOTPError creates a new OTP error
func NewOTPError(code string, message string) *OTPError {
	return &OTPError{Code: code, Message: message}
}
