package model

type RequestOTPRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
}

type VerifyOTPRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	OTP         string `json:"otp" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AuthUserResponse struct {
	ID            string `json:"id"`
	PhoneNumber   string `json:"phone_number"`
	PhoneVerified bool   `json:"phone_verified"`
}

type AuthResponse struct {
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token"`
	ExpiresIn    int              `json:"expires_in"`
	IsNewUser    bool             `json:"is_new_user"`
	User         AuthUserResponse `json:"user"`
}

type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}
