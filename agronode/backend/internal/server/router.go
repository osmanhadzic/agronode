package server

import (
	"log/slog"

	"agronode/backend/internal/handlers"
	"github.com/gin-gonic/gin"
)

func NewRouter(logger *slog.Logger, telemetryService handlers.TelemetryQueryService) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery(), gin.Logger())

	api := router.Group("/api")
	handlers.RegisterHealthRoutes(api, logger)
	handlers.RegisterTelemetryRoutes(api, logger, telemetryService)

	return router
}
