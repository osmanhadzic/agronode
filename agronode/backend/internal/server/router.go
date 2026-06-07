package server

import (
	"log/slog"
	"net/http"

	"agronode/backend/internal/handlers"
	"agronode/backend/internal/realtime"
	"github.com/gin-gonic/gin"
)

func NewRouter(logger *slog.Logger, telemetryService handlers.TelemetryQueryService, deviceService handlers.DeviceRegistrationService, realtimeHub *realtime.Hub) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery(), gin.Logger())
	router.Use(corsMiddleware())

	api := router.Group("/api")
	handlers.RegisterHealthRoutes(api, logger)
	handlers.RegisterTelemetryRoutes(api, logger, telemetryService)
	handlers.RegisterDeviceRoutes(api, logger, deviceService)
	handlers.RegisterRealtimeRoutes(router, logger, realtimeHub)

	return router
}

func corsMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		origin := context.GetHeader("Origin")
		if origin != "" {
			context.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			context.Writer.Header().Set("Vary", "Origin")
		}

		context.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		context.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		context.Writer.Header().Set("Access-Control-Max-Age", "43200")

		if context.Request.Method == http.MethodOptions {
			context.AbortWithStatus(http.StatusNoContent)
			return
		}

		context.Next()
	}
}
