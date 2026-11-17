package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/gateway/internal/response"
	authv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/auth/v1"
	userv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthHandler struct {
	authClient authv1.AuthServiceClient
	userClient userv1.UserServiceClient
}

func NewAuthHandler(
	authClient authv1.AuthServiceClient,
	userClient userv1.UserServiceClient,
) *AuthHandler {

	return &AuthHandler{
		authClient: authClient,
		userClient: userClient,
	}
}

type RequestOTPRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
}

type VerifyOTPRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	OTP         string `json:"otp" binding:"required"`
}

// RequestOTPResponseData is the documented shape of the "data" field
// returned by POST /auth/otp/request (built via gin.H at runtime).
type RequestOTPResponseData struct {
	Message string `json:"message"`
}

// VerifyOTPResponseUser is the authenticated user summary nested in the
// VerifyOTP response.
type VerifyOTPResponseUser struct {
	ID            string `json:"id"`
	PhoneNumber   string `json:"phone_number"`
	PhoneVerified bool   `json:"phone_verified"`
}

// VerifyOTPResponseData is the documented shape of the "data" field
// returned by POST /auth/otp/verify (built via gin.H at runtime).
type VerifyOTPResponseData struct {
	AccessToken         string                `json:"access_token"`
	RefreshToken        string                `json:"refresh_token"`
	IsNewUser           bool                  `json:"is_new_user"`
	OnboardingCompleted bool                  `json:"onboarding_completed"`
	User                VerifyOTPResponseUser `json:"user"`
}

// RequestOTP godoc
//
//	@Summary		Request an OTP
//	@Description	Sends a one-time password via SMS to the given phone number. Used to start login/signup.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RequestOTPRequest							true	"Phone number to send the OTP to"
//	@Success		200		{object}	response.SuccessResponse{data=RequestOTPResponseData}
//	@Failure		400		{object}	response.ErrorResponse
//	@Failure		429		{object}	response.ErrorResponse	"Too many OTP requests for this phone number"
//	@Failure		500		{object}	response.ErrorResponse
//	@Router			/auth/otp/request [post]
func (h *AuthHandler) RequestOTP(
	c *gin.Context,
) {

	var request RequestOTPRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid request", nil)

		return
	}

	ctx, cancel :=
		context.WithTimeout(
			c.Request.Context(),
			3*time.Second,
		)

	defer cancel()

	otpResp, err :=
		h.authClient.RequestOTP(
			ctx,
			&authv1.RequestOTPRequest{
				PhoneNumber: request.PhoneNumber,
			},
		)

	if err != nil {
		if status.Code(err) == codes.ResourceExhausted {
			response.TooManyRequests(c, status.Convert(err).Message(), nil)
			return
		}
		response.Internal(c, "authentication service unavailable", nil)

		return
	}

	response.Success(
		c,
		http.StatusOK,
		gin.H{
			"message": otpResp.Message,
		},
	)
}

// VerifyOTP godoc
//
//	@Summary		Verify an OTP and log in
//	@Description	Verifies the OTP sent to the given phone number. On success returns an access/refresh token pair
//	@Description	and the authenticated user's profile completion state. Creates the user on first verification.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		VerifyOTPRequest	true	"Phone number and OTP code to verify"
//	@Success		200		{object}	response.SuccessResponse{data=VerifyOTPResponseData}
//	@Failure		400		{object}	response.ErrorResponse
//	@Failure		401		{object}	response.ErrorResponse	"OTP is invalid or expired"
//	@Failure		500		{object}	response.ErrorResponse
//	@Router			/auth/otp/verify [post]
func (h *AuthHandler) VerifyOTP(
	c *gin.Context,
) {
	var request VerifyOTPRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid request", nil)

		return
	}

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		3*time.Second,
	)
	defer cancel()

	verifyResp, err := h.authClient.VerifyOTP(
		ctx,
		&authv1.VerifyOTPRequest{
			PhoneNumber: request.PhoneNumber,
			Otp:         request.OTP,
		},
	)

	if err != nil {
		response.Unauthorized(c, "invalid or expired OTP", nil)

		return
	}
	if verifyResp == nil || verifyResp.User == nil || verifyResp.User.Id == "" {
		log.Printf("verify OTP returned an invalid authenticated user: request_id=%s", c.GetString(response.RequestIDContextKey))
		response.Internal(c, "unable to load user profile", nil)

		return
	}

	profileResponse, err := h.userClient.GetProfile(ctx, &userv1.GetProfileRequest{
		UserId: verifyResp.User.Id,
	})
	if err != nil {
		log.Printf("get profile failed: request_id=%s user_id=%s error=%v", c.GetString(response.RequestIDContextKey), verifyResp.User.Id, err)
		response.Internal(c, "unable to load user profile", nil)
		return
	}
	if profileResponse == nil || profileResponse.Profile == nil {
		log.Printf("get profile returned an empty response: request_id=%s user_id=%s", c.GetString(response.RequestIDContextKey), verifyResp.User.Id)
		response.Internal(c, "unable to load user profile", nil)

		return
	}

	response.Success(
		c,
		http.StatusOK,
		gin.H{
			"access_token":  verifyResp.AccessToken,
			"refresh_token": verifyResp.RefreshToken,
			"is_new_user":   verifyResp.IsNewUser,
			// "profile_completed": profileResponse.Profile.ProfileCompleted,
			"onboarding_completed": profileResponse.Profile.OnboardingCompleted,
			"user": gin.H{
				"id":             verifyResp.User.Id,
				"phone_number":   verifyResp.User.PhoneNumber,
				"phone_verified": verifyResp.User.PhoneVerified,
			},
		},
	)
}
