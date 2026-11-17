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
	JWTSecret   string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:      os.Getenv("APP_ENV"),
		Port:        os.Getenv("HTTP_PORT"),
		GRPCPort:    os.Getenv("GRPC_PORT"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}

	if cfg.Port == "" {
		cfg.Port = "8082"
	}

	if cfg.GRPCPort == "" {
		cfg.GRPCPort = "50053"
	}

	// Platforms like Cloud Run inject a reserved PORT env var and only route
	// external traffic to a single port per container. Only the gRPC server
	// needs to be reachable from outside the process in that environment
	// (the gateway is the only HTTP-facing service), so prefer PORT over
	// GRPC_PORT whenever it is set. Local/dev setups that don't set PORT are
	// unaffected.
	if envPort := os.Getenv("PORT"); envPort != "" {
		cfg.GRPCPort = envPort
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}
