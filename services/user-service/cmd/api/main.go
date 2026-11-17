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

	userv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/user/v1"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/config"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/database"
	grpcserver "github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/grpc"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/handler"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/repository"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/routes"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/service"
)

func main() {
	ctx := context.Background()

	// -------------------------
	// Configuration
	// -------------------------

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(
			"failed to load configuration: ",
			err,
		)
	}

	// -------------------------
	// Database connection
	// -------------------------

	db, err := database.NewPostgresPool(
		ctx,
		cfg.DatabaseURL,
	)
	if err != nil {
		log.Fatal(
			"failed to connect database: ",
			err,
		)
	}

	defer db.Close()

	// -------------------------
	// Repositories
	// -------------------------

	profileRepository :=
		repository.NewProfileRepository(db)

	ownerRepository :=
		repository.NewOwnerRepository(db)

	businessTypeRepository :=
		repository.NewBusinessTypeRepository(db)

	workTypeRepository :=
		repository.NewWorkTypeRepository(db)

	workCategoryRepository :=
		repository.NewWorkCategoryRepository(db)

	workSegmentRepository :=
		repository.NewWorkSegmentRepository(db)

	workTypeFieldRepository :=
		repository.NewWorkTypeFieldRepository(db)

	rateTypeRepository :=
		repository.NewRateTypeRepository(db)

	skillRepository :=
		repository.NewSkillRepository(db)

	commonRoleRepository :=
		repository.NewCommonRoleRepository(db)

	// -------------------------
	// Services
	// -------------------------

	ownerService :=
		service.NewOwnerService(
			ownerRepository,
			businessTypeRepository,
		)

	profileService :=
		service.NewProfileService(
			profileRepository,
			ownerService,
			workTypeRepository,
			workCategoryRepository,
			workSegmentRepository,
			workTypeFieldRepository,
			rateTypeRepository,
			skillRepository,
			commonRoleRepository,
		)

	userService :=
		service.NewUserService(
			profileService,
		)

	// -------------------------
	// HTTP / Gin
	// -------------------------

	profileHandler :=
		handler.NewProfileHandler(
			profileService,
		)

	ownerHandler :=
		handler.NewOwnerHandler(
			ownerService,
		)

	router := gin.Default()

	routes.RegisterRoutes(
		router,
		profileHandler,
		ownerHandler,
		cfg.JWTSecret,
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
			"failed to listen on gRPC port "+cfg.GRPCPort+": ",
			err,
		)
	}

	grpcServer := grpc.NewServer()

	userGRPCServer :=
		grpcserver.NewUserServer(
			userService,
		)

	userv1.RegisterUserServiceServer(
		grpcServer,
		userGRPCServer,
	)

	// Start gRPC server concurrently
	go func() {
		log.Println(
			"user-service gRPC listening on :" + cfg.GRPCPort,
		)

		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal(
				"user-service gRPC server failed: ",
				err,
			)
		}
	}()

	// -------------------------
	// HTTP server
	// -------------------------

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf(
			"user-service HTTP listening on :%s",
			cfg.Port,
		)

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(
				"user-service HTTP server failed: ",
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

	log.Println("user-service shutting down...")

	httpShutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(httpShutdownCtx); err != nil {
		log.Printf("user-service HTTP shutdown error: %v", err)
	}

	grpcServer.GracefulStop()
}
