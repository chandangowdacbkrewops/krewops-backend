package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv   string
	Port     string
	GRPCPort string

	DatabaseURL string

	JWTSecret              string
	JWTAccessExpiryMinutes int

	OTPExpiryMinutes           int
	OTPResendMinimumSeconds    int
	OTPMaxRequests             int
	OTPMaxRequestWindowMinutes int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	jwtExpiry, err := getInt("JWT_ACCESS_EXPIRY_MINUTES", 60)
	if err != nil {
		return nil, err
	}

	otpExpiry, err := getInt("OTP_EXPIRY_MINUTES", 5)
	if err != nil {
		return nil, err
	}

	otpResendMinSeconds, err := getInt("OTP_RESEND_MINIMUM_SECONDS", 60)
	if err != nil {
		return nil, err
	}

	otpMaxRequests, err := getInt("OTP_MAX_REQUESTS", 3)
	if err != nil {
		return nil, err
	}

	otpMaxRequestWindow, err := getInt("OTP_MAX_REQUEST_WINDOW_MINUTES", 15)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		AppEnv:   os.Getenv("APP_ENV"),
		Port:     os.Getenv("HTTP_PORT"),
		GRPCPort: os.Getenv("GRPC_PORT"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		JWTSecret:              os.Getenv("JWT_SECRET"),
		JWTAccessExpiryMinutes: jwtExpiry,

		OTPExpiryMinutes:         otpExpiry,
		OTPResendMinimumSeconds:  otpResendMinSeconds,
		OTPMaxRequests:           otpMaxRequests,
		OTPMaxRequestWindowMinutes: otpMaxRequestWindow,
	}

	if cfg.Port == "" {
		cfg.Port = "8082"
	}

	if cfg.GRPCPort == "" {
		cfg.GRPCPort = "50051"
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getInt(key string, defaultValue int) (int, error) {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number", key)
	}

	return parsed, nil
}
