package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"agronode/backend/internal/models"
	"agronode/backend/internal/services"
	"github.com/gin-gonic/gin"
)

const deviceAPIKeyHeader = "X-Device-API-Key"

type DeviceConnectionService interface {
	Connect(ctx context.Context, deviceID, apiKey, certificateSerial string) (*models.Device, error)
	Heartbeat(ctx context.Context, deviceID, apiKey, certificateSerial string) (*models.Device, error)
	Disconnect(ctx context.Context, deviceID, apiKey, certificateSerial string) (*models.Device, error)
}

type deviceConnectionHandler struct {
	logger  *slog.Logger
	service DeviceConnectionService
}

type deviceConnectionAction func(context.Context, string, string, string) (*models.Device, error)

func RegisterDeviceConnectionRoutes(api *gin.RouterGroup, logger *slog.Logger, service DeviceConnectionService) {
	handler := &deviceConnectionHandler{logger: logger, service: service}
	devices := api.Group("/devices")
	devices.POST("/:deviceId/connect", handler.connect)
	devices.POST("/:deviceId/heartbeat", handler.heartbeat)
	devices.POST("/:deviceId/disconnect", handler.disconnect)
}

func (handler *deviceConnectionHandler) connect(ctx *gin.Context) {
	handler.handle(ctx, handler.service.Connect)
}

func (handler *deviceConnectionHandler) heartbeat(ctx *gin.Context) {
	handler.handle(ctx, handler.service.Heartbeat)
}

func (handler *deviceConnectionHandler) disconnect(ctx *gin.Context) {
	handler.handle(ctx, handler.service.Disconnect)
}

func (handler *deviceConnectionHandler) handle(ctx *gin.Context, action deviceConnectionAction) {
	var certificateSerial string
	if requestTLS := ctx.Request.TLS; requestTLS != nil && len(requestTLS.VerifiedChains) > 0 && len(requestTLS.PeerCertificates) > 0 {
		certificateSerial = requestTLS.PeerCertificates[0].SerialNumber.Text(10)
	}

	device, err := action(ctx.Request.Context(), ctx.Param("deviceId"), ctx.GetHeader(deviceAPIKeyHeader), certificateSerial)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrDeviceValidation):
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid device id"})
		case errors.Is(err, services.ErrDeviceAuthentication):
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "device authentication failed"})
		case errors.Is(err, services.ErrDeviceNotProvisioned),
			errors.Is(err, services.ErrDeviceInactive),
			errors.Is(err, services.ErrDeviceCertificateMismatch),
			errors.Is(err, services.ErrDeviceCertificateExpired):
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			if handler.logger != nil {
				handler.logger.Error("device connection update failed", "deviceId", ctx.Param("deviceId"), "error", err)
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "device connection update failed"})
		}
		return
	}

	response := gin.H{"deviceId": device.DeviceID, "status": device.Status}
	if device.LastSeen != nil {
		response["lastSeen"] = device.LastSeen.UTC().Format(time.RFC3339Nano)
	}
	ctx.JSON(http.StatusOK, response)
}

