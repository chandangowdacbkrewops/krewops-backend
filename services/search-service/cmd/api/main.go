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

	searchv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/search/v1"

	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/config"
	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/database"
	grpcserver "github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/grpc"
	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/handler"
	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/repository"
	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/routes"
	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/service"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load configuration: ", err)
	}

	// search-service owns no tables. Worker search reads PostgreSQL tables
	// owned by user-service; work search reads the Mongo work_postings
	// collection owned by work-service.
	db, err := database.NewPostgresPool(
		ctx,
		cfg.DatabaseURL,
	)
	if err != nil {
		log.Fatal("failed to connect database: ", err)
	}

	defer db.Close()

	mongoClient, mongoDB, err := database.NewMongoDatabase(
		ctx,
		cfg.MongoURI,
		cfg.MongoDB,
	)
	if err != nil {
		log.Fatal("failed to connect mongo: ", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Printf("failed to disconnect mongo: %v", err)
		}
	}()

	// -------------------------
	// Repositories
	// -------------------------

	workSearchRepository := repository.NewWorkSearchRepository(mongoDB)
	workerSearchRepository := repository.NewWorkerSearchRepository(db)

	// -------------------------
	// Services
	// -------------------------

	searchService := service.NewSearchService(
		workSearchRepository,
		workerSearchRepository,
	)

	// -------------------------
	// HTTP / Gin
	// -------------------------

	searchHandler := handler.NewSearchHandler(searchService)

	router := gin.Default()

	routes.RegisterRoutes(
		router,
		searchHandler,
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
			"failed to listen on gRPC port: ",
			err,
		)
	}

	grpcServer := grpc.NewServer()

	searchv1.RegisterSearchServiceServer(
		grpcServer,
		grpcserver.NewSearchServer(searchService),
	)

	go func() {
		log.Println(
			"search-service gRPC listening on :" + cfg.GRPCPort,
		)

		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal(
				"gRPC server failed: ",
				err,
			)
		}
	}()

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf(
			"search-service HTTP listening on :%s",
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

	log.Println("search-service shutting down...")

	httpShutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(httpShutdownCtx); err != nil {
		log.Printf("search-service HTTP shutdown error: %v", err)
	}

	grpcServer.GracefulStop()
}
