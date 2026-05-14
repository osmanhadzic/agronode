package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agronode/backend/internal/config"
	"agronode/backend/internal/database"
	"agronode/backend/internal/mqtt"
	"agronode/backend/internal/realtime"
	"agronode/backend/internal/repositories"
	"agronode/backend/internal/server"
	"agronode/backend/internal/services"
)

func main() {
	cfg := config.Load()
	logger := config.NewLogger(cfg.LogLevel)

	startupContext, startupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer startupCancel()

	databaseConnection, err := database.ConnectPostgres(startupContext, cfg, logger)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}

	if err := database.RunMigrations(startupContext, databaseConnection, logger); err != nil {
		logger.Error("database migration failed", "error", err)
		os.Exit(1)
	}

	rawDatabase, err := databaseConnection.DB()
	if err != nil {
		logger.Error("database handle failed", "error", err)
		os.Exit(1)
	}
	defer rawDatabase.Close()

	store := database.NewStore(databaseConnection)
	telemetryRepository := repositories.NewGormTelemetryRepository(store.DB)
	telemetryService := services.NewTelemetryService(telemetryRepository, logger)
	realtimeHub := realtime.NewHub(logger)
	telemetryService.SetBroadcaster(realtimeHub)

	appContext, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	mqttClient := mqtt.NewClient(cfg.MQTTBroker, cfg.MQTTTopic, logger, telemetryService)
	mqttErrorChannel := make(chan error, 1)

	go func() {
		if err := mqttClient.Run(appContext); err != nil {
			mqttErrorChannel <- err
		}
	}()

	router := server.NewRouter(logger, telemetryService, realtimeHub)
	httpServer := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("http server starting", "port", cfg.AppPort)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)

	select {
	case <-shutdownSignals:
		logger.Info("shutdown signal received")
	case err := <-mqttErrorChannel:
		logger.Error("mqtt client failed", "error", err)
	}

	appCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("http server stopped")
}
