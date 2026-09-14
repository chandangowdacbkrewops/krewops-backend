package grpcserver

import (
	"context"

	authv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/auth/v1"
	"github.com/chandangowdacbkrewops/krewops-backend/services/auth-service/internal/service"
)

type AuthServer struct {
	authv1.UnimplementedAuthServiceServer

	authService *service.AuthService
}

func NewAuthServer(
	authService *service.AuthService,
) *AuthServer {

	return &AuthServer{
		authService: authService,
	}
}

func (s *AuthServer) RequestOTP(
	ctx context.Context,
	req *authv1.RequestOTPRequest,
) (*authv1.RequestOTPResponse, error) {

	err := s.authService.RequestOTP(
		ctx,
		req.PhoneNumber,
	)

	if err != nil {
		return nil, err
	}

	return &authv1.RequestOTPResponse{
		Message: "OTP sent successfully",
	}, nil
}

func (s *AuthServer) VerifyOTP(
	ctx context.Context,
	req *authv1.VerifyOTPRequest,
) (*authv1.VerifyOTPResponse, error) {

	result, err := s.authService.VerifyOTP(
		ctx,
		req.PhoneNumber,
		req.Otp,
	)

	if err != nil {
		return nil, err
	}

	return &authv1.VerifyOTPResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		IsNewUser:    result.IsNewUser,
		User: &authv1.AuthUser{
			Id:            result.User.ID,
			PhoneNumber:   result.User.PhoneNumber,
			PhoneVerified: result.User.PhoneVerified,
		},
	}, nil
}
