package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

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
	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf(
			"auth-service HTTP listening on :%s",
			cfg.Port,
		)

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(
				"HTTP server failed: ",
				err,
			)
		}
	}()

	// Wait for SIGINT/SIGTERM (Cloud Run sends SIGTERM on scale-down or
	// redeploy) and shut down both servers cleanly instead of being killed
	// mid-request.
	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-shutdownCtx.Done()

	log.Println("auth-service shutting down...")

	httpShutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(httpShutdownCtx); err != nil {
		log.Printf("auth-service HTTP shutdown error: %v", err)
	}

	grpcServer.GracefulStop()
}
