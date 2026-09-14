package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv   string
	Port     string
	GRPCPort string

	DatabaseURL string

	JWTSecret string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:   os.Getenv("APP_ENV"),
		Port:     os.Getenv("HTTP_PORT"),
		GRPCPort: os.Getenv("GRPC_PORT"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		JWTSecret: os.Getenv("JWT_SECRET"),
	}

	if cfg.Port == "" {
		cfg.Port = "8085"
	}

	if cfg.GRPCPort == "" {
		cfg.GRPCPort = "50054"
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}
