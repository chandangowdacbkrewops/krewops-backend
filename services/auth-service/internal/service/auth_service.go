package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"

	"github.com/chandangowdacbkrewops/krewops-backend/services/auth-service/internal/config"
	"github.com/chandangowdacbkrewops/krewops-backend/services/auth-service/internal/model"
	"github.com/chandangowdacbkrewops/krewops-backend/services/auth-service/internal/repository"
)

const refreshTokenExpiryDays = 30

type SMSProvider interface {
	SendOTP(ctx context.Context, phone string, otp string) error
}

type DevSMSProvider struct{}

func (p *DevSMSProvider) SendOTP(_ context.Context, phone string, otp string) error {
	log.Printf("[DEV] OTP for %s = %s", phone, otp)
	return nil
}

type TokenManager struct {
	secret              string
	accessExpiryMinutes int
}

func NewTokenManager(cfg *config.Config) *TokenManager {
	return &TokenManager{
		secret:              cfg.JWTSecret,
		accessExpiryMinutes: cfg.JWTAccessExpiryMinutes,
	}
}

func (tm *TokenManager) GenerateAccessToken(userID string) (string, int, error) {
	expiresIn := tm.accessExpiryMinutes * 60
	expiresAt := time.Now().Add(time.Duration(tm.accessExpiryMinutes) * time.Minute)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": expiresAt.Unix(),
		"iat": time.Now().Unix(),
	})

	signed, err := token.SignedString([]byte(tm.secret))
	if err != nil {
		return "", 0, err
	}

	return signed, expiresIn, nil
}

func (tm *TokenManager) GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

type AuthService struct {
	authRepository             *repository.AuthRepository
	smsProvider                SMSProvider
	tokenManager               *TokenManager
	otpExpiryMinutes           int
	otpResendMinimumSeconds    int
	otpMaxRequests             int
	otpMaxRequestWindowMinutes int
}

func NewAuthService(
	authRepository *repository.AuthRepository,
	smsProvider SMSProvider,
	tokenManager *TokenManager,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		authRepository:             authRepository,
		smsProvider:                smsProvider,
		tokenManager:               tokenManager,
		otpExpiryMinutes:           cfg.OTPExpiryMinutes,
		otpResendMinimumSeconds:    cfg.OTPResendMinimumSeconds,
		otpMaxRequests:             cfg.OTPMaxRequests,
		otpMaxRequestWindowMinutes: cfg.OTPMaxRequestWindowMinutes,
	}
}

func (s *AuthService) FindOrCreateUser(
	ctx context.Context,
	phoneNumber string,
) (*model.User, bool, error) {
	user, err := s.authRepository.FindUserByPhone(ctx, phoneNumber)
	if err != nil {
		return nil, false, err
	}

	if user != nil {
		if !user.PhoneVerified {
			if err := s.authRepository.MarkPhoneVerified(ctx, user.ID); err != nil {
				return nil, false, err
			}
			user.PhoneVerified = true
		}
		return user, false, nil
	}

	user, err = s.authRepository.CreateUser(ctx, phoneNumber)
	if err != nil {
		return nil, false, err
	}

	return user, true, nil
}

func (s *AuthService) RequestOTP(ctx context.Context, phone string) error {
	// Check if user has exceeded maximum OTP requests in the time window
	requestCount, err := s.authRepository.GetOTPRequestCount(ctx, phone, s.otpMaxRequestWindowMinutes)
	if err != nil {
		return fmt.Errorf("failed to check OTP request count: %w", err)
	}

	if requestCount >= s.otpMaxRequests {
		return repository.ErrOTPRateLimited
	}

	// Check if last OTP request was too recent (resend throttling)
	lastRequestTime, err := s.authRepository.GetLastOTPRequestTime(ctx, phone)
	if err != nil {
		return fmt.Errorf("failed to check last OTP request time: %w", err)
	}

	if !lastRequestTime.IsZero() {
		timeSinceLastRequest := time.Since(lastRequestTime).Seconds()
		if timeSinceLastRequest < float64(s.otpResendMinimumSeconds) {
			return repository.ErrOTPResendTooSoon
		}
	}

	// Generate OTP
	otp, err := generateOTP()
	if err != nil {
		return err
	}

	otpHash := HashToken(otp)
	expiresAt := time.Now().Add(time.Duration(s.otpExpiryMinutes) * time.Minute)

	// Save OTP (this will invalidate previous OTPs)
	if err := s.authRepository.SaveOTP(ctx, phone, otpHash, expiresAt); err != nil {
		return err
	}

	// Send OTP via SMS
	return s.smsProvider.SendOTP(ctx, phone, otp)
}

func (s *AuthService) VerifyOTP(
	ctx context.Context,
	phone string,
	otp string,
) (*model.AuthResponse, error) {
	otpHash := HashToken(otp)

	valid, err := s.authRepository.VerifyOTP(ctx, phone, otpHash)
	if err != nil {
		return nil, err
	}

	if !valid {
		return nil, fmt.Errorf("invalid or expired OTP")
	}

	user, isNewUser, err := s.FindOrCreateUser(ctx, phone)
	if err != nil {
		return nil, err
	}

	accessToken, expiresIn, err := s.tokenManager.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenManager.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	refreshHash := HashToken(refreshToken)
	refreshExpiry := time.Now().Add(refreshTokenExpiryDays * 24 * time.Hour)

	if err := s.authRepository.SaveRefreshToken(ctx, user.ID, refreshHash, refreshExpiry); err != nil {
		return nil, err
	}

	return &model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		IsNewUser:    isNewUser,
		User: model.AuthUserResponse{
			ID:            user.ID,
			PhoneNumber:   user.PhoneNumber,
			PhoneVerified: user.PhoneVerified,
		},
	}, nil
}

func (s *AuthService) RefreshToken(
	ctx context.Context,
	refreshToken string,
) (*model.RefreshTokenResponse, error) {
	tokenHash := HashToken(refreshToken)

	userID, expiresAt, revokedAt, err := s.authRepository.FindRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("invalid refresh token")
		}
		return nil, err
	}

	if revokedAt != nil {
		return nil, fmt.Errorf("refresh token has been revoked")
	}

	if time.Now().After(expiresAt) {
		return nil, fmt.Errorf("refresh token has expired")
	}

	user, err := s.authRepository.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	accessToken, expiresIn, err := s.tokenManager.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &model.RefreshTokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   expiresIn,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := HashToken(refreshToken)
	return s.authRepository.RevokeRefreshToken(ctx, tokenHash)
}

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
