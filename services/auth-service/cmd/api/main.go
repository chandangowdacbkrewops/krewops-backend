package main

import (
	"context"
	"log"
	"net"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	authv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/auth/v1"

	"github.com/chandangowdacbkrewops/krewops-backend/services/auth-service/internal/config"
	"github.com/chandangowdacbkrewops/krewops-backend/services/auth-service/internal/database"
	grpcserver "github.com/chandangowdacbkrewops/krewops-backend/services/auth-service/internal/grpc"
	"github.com/chandangowdacbkrewops/krewops-backend/services/auth-service/internal/handler"
	"github.com/chandangowdacbkrewops/krewops-backend/services/auth-service/internal/repository"
	"github.com/chandangowdacbkrewops/krewops-backend/services/auth-service/internal/routes"
	"github.com/chandangowdacbkrewops/krewops-backend/services/auth-service/internal/service"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load configuration: ", err)
	}

	if err := database.RunMigrations(
		cfg.DatabaseURL,
		"migrations",
		"auth_schema_migrations",
	); err != nil {
		log.Fatal("failed to run migrations: ", err)
	}

	db, err := database.NewPostgresPool(
		ctx,
		cfg.DatabaseURL,
	)
	if err != nil {
		log.Fatal("failed to connect database: ", err)
	}

	defer db.Close()

	// Dependencies
	authRepository := repository.NewAuthRepository(db)

	tokenManager := service.NewTokenManager(cfg)

	smsProvider := &service.DevSMSProvider{}

	authService := service.NewAuthService(
		authRepository,
		smsProvider,
		tokenManager,
		cfg,
	)

	// -------------------------
	// HTTP / Gin
	// -------------------------

	authHandler := handler.NewAuthHandler(
		authService,
	)

	router := gin.Default()

	routes.RegisterRoutes(
		router,
		authHandler,
	)

	// -------------------------
	// gRPC
	// -------------------------

	listener, err := net.Listen(
		"tcp",
		":"+cfg.GRPCPort,
	)
	if err != nil {
		log.Fatal(
			"failed to listen on gRPC port: ",
			err,
		)
	}

	grpcServer := grpc.NewServer()

	authv1.RegisterAuthServiceServer(
		grpcServer,
		grpcserver.NewAuthServer(
			authService,
		),
	)

	// Start gRPC concurrently
	go func() {
		log.Println(
			"auth-service gRPC listening on :" + cfg.GRPCPort,
		)

		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal(
				"gRPC server failed: ",
				err,
			)
		}
	}()

	// Start HTTP server
	log.Printf(
		"auth-service HTTP listening on :%s",
		cfg.Port,
	)

	if err := router.Run(
		":" + cfg.Port,
	); err != nil {
		log.Fatal(
			"HTTP server failed: ",
			err,
		)
	}
}
