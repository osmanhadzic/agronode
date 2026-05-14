package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type healthHandler struct {
	logger *slog.Logger
}

type healthResponse struct {
	Status string `json:"status"`
}

func RegisterHealthRoutes(api *gin.RouterGroup, logger *slog.Logger) {
	handler := &healthHandler{logger: logger}
	api.GET("/health", handler.getHealth)
}

func (handler *healthHandler) getHealth(context *gin.Context) {
	handler.logger.Debug("health check requested")
	context.JSON(http.StatusOK, healthResponse{Status: "ok"})
}
