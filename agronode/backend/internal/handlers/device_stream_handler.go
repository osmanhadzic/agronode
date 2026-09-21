package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"agronode/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type DeviceStreamControlService interface {
	SetTelemetryStreaming(ctx context.Context, deviceID string, sensorID string, enabled bool) error
}

type deviceStreamHandler struct {
	logger  *slog.Logger
	service DeviceStreamControlService
}

type deviceStreamControlRequest struct {
	Action   string `json:"action"`
	SensorID string `json:"sensorId"`
}

type deviceStreamControlResponse struct {
	DeviceID  string `json:"deviceId"`
	SensorID  string `json:"sensorId"`
	Action    string `json:"action"`
	Streaming bool   `json:"streaming"`
}

func RegisterDeviceStreamRoutes(api *gin.RouterGroup, logger *slog.Logger, service DeviceStreamControlService) {
	handler := &deviceStreamHandler{logger: logger, service: service}

	api.POST("/devices/:deviceId/stream-control", handler.setStreamControl)
}

func (handler *deviceStreamHandler) setStreamControl(context *gin.Context) {
	deviceID := context.Param("deviceId")
	requestContext, err := requestContextWithOrganizationScope(context)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var request deviceStreamControlRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	action := strings.ToLower(strings.TrimSpace(request.Action))
	if action != "pause" && action != "resume" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "action must be pause or resume"})
		return
	}

	enabled := action == "resume"
	trimmedSensorID := strings.TrimSpace(request.SensorID)
	if trimmedSensorID == "" {
		trimmedSensorID = "all"
	}
	if err := handler.service.SetTelemetryStreaming(requestContext, deviceID, trimmedSensorID, enabled); err != nil {
		if errors.Is(err, services.ErrValidation) {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if handler.logger != nil {
			handler.logger.Error("set device stream control failed", "deviceId", deviceID, "action", action, "error", err)
		}

		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set device stream control"})
		return
	}

	context.JSON(http.StatusOK, deviceStreamControlResponse{
		DeviceID:  deviceID,
		SensorID:  trimmedSensorID,
		Action:    action,
		Streaming: enabled,
	})
}
