package server

import (
	"log/slog"
	"net/http"
	"time"

	"agronode/backend/internal/handlers"
	"agronode/backend/internal/realtime"
	"agronode/backend/internal/services"
	"github.com/gin-gonic/gin"
)

func NewRouter(logger *slog.Logger, sessionSecret string, authService handlers.AuthService, telemetryService handlers.TelemetryQueryService, streamControlService handlers.DeviceStreamControlService, triggerService handlers.TriggerService, deviceService handlers.DeviceRegistrationService, organizationService handlers.OrganizationService, realtimeHub *realtime.Hub, adminChatLLM services.AdminChatLLMService) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery(), requestLoggerMiddleware(logger))
	router.Use(corsMiddleware())

	publicAPI := router.Group("/api")
	handlers.RegisterHealthRoutes(publicAPI, logger)
	handlers.RegisterAuthRoutes(publicAPI, logger, authService)

	router.Use(sessionAuthMiddleware(sessionSecret))

	api := router.Group("/api")
	handlers.RegisterTelemetryRoutes(api, logger, telemetryService)
	handlers.RegisterDeviceRoutes(api, logger, deviceService)
	handlers.RegisterOrganizationRoutes(api, logger, organizationService)
	handlers.RegisterDeviceStreamRoutes(api, logger, streamControlService)
	handlers.RegisterTriggerRoutes(api, logger, triggerService)
	handlers.RegisterRealtimeRoutes(router, logger, realtimeHub)
	handlers.RegisterAdminChatRoutes(router, logger, adminChatLLM)

	return router
}

func requestLoggerMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(context *gin.Context) {
		startedAt := time.Now()
		path := context.Request.URL.Path
		query := context.Request.URL.RawQuery

		context.Next()

		latency := time.Since(startedAt)
		statusCode := context.Writer.Status()
		fields := []any{
			"method", context.Request.Method,
			"path", path,
			"query", query,
			"status", statusCode,
			"latencyMs", latency.Milliseconds(),
			"clientIp", context.ClientIP(),
			"userAgent", context.Request.UserAgent(),
			"bytes", context.Writer.Size(),
		}

		switch {
		case statusCode >= http.StatusInternalServerError:
			logger.Error("http request completed", fields...)
		case statusCode >= http.StatusBadRequest:
			logger.Warn("http request completed", fields...)
		default:
			logger.Info("http request completed", fields...)
		}
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		origin := context.GetHeader("Origin")
		if origin != "" {
			context.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			context.Writer.Header().Set("Vary", "Origin")
		}

		context.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		context.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Organization-ID, X-Frontend-Token")
		context.Writer.Header().Set("Access-Control-Max-Age", "43200")

		if context.Request.Method == http.MethodOptions {
			context.AbortWithStatus(http.StatusNoContent)
			return
		}

		context.Next()
	}
}
