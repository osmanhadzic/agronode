package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"agronode/backend/internal/models"
	"agronode/backend/internal/repositories"
	"agronode/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type DeviceRegistrationService interface {
	RegisterDevice(ctx context.Context, deviceID string, firmwareVersion string, metadata models.DeviceMetadata, apiKey string, provisioningToken string, tags []string) (*models.Device, error)
	GetDevice(ctx context.Context, deviceID string) (*models.Device, error)
	ListDevices(ctx context.Context, params services.DeviceListParams) ([]models.Device, error)
}

type deviceHandler struct {
	logger  *slog.Logger
	service DeviceRegistrationService
}

type registerDeviceRequest struct {
	DeviceID          string                `json:"deviceId" binding:"required"`
	FirmwareVersion   string                `json:"firmwareVersion,omitempty"`
	Metadata          models.DeviceMetadata `json:"metadata,omitempty"`
	APIKey            string                `json:"apiKey,omitempty"`
	ProvisioningToken string                `json:"provisioningToken,omitempty"`
	Tags              []string              `json:"tags,omitempty"`
}

type deviceResponse struct {
	ID              uint                     `json:"id"`
	DeviceID        string                   `json:"deviceId"`
	Status          string                   `json:"status"`
	FirmwareVersion string                   `json:"firmwareVersion,omitempty"`
	Metadata        models.DeviceMetadata    `json:"metadata,omitempty"`
	Tags            []string                 `json:"tags,omitempty"`
	DesiredState    models.DeviceShadowState `json:"desiredState,omitempty"`
	ReportedState   models.DeviceShadowState `json:"reportedState,omitempty"`
	LastSeen        string                   `json:"lastSeen,omitempty"`
	CreatedAt       string                   `json:"createdAt"`
	UpdatedAt       string                   `json:"updatedAt"`
}

func RegisterDeviceRoutes(api *gin.RouterGroup, logger *slog.Logger, service DeviceRegistrationService) {
	handler := &deviceHandler{logger: logger, service: service}

	devices := api.Group("/devices")
	devices.POST("/register", handler.registerDevice)
	devices.GET("/:deviceId", handler.getDevice)
	devices.GET("", handler.listDevices)
}

func (handler *deviceHandler) registerDevice(ctx *gin.Context) {
	var req registerDeviceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		handler.logger.Warn("invalid registration request", "error", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	device, err := handler.service.RegisterDevice(ctx.Request.Context(), req.DeviceID, req.FirmwareVersion, req.Metadata, req.APIKey, req.ProvisioningToken, req.Tags)
	if err != nil {
		handler.logger.Error("device registration failed", "deviceId", req.DeviceID, "error", err)

		if errors.Is(err, services.ErrDeviceValidation) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register device"})
		return
	}

	ctx.JSON(http.StatusOK, toDeviceResponse(device))
}

func (handler *deviceHandler) getDevice(ctx *gin.Context) {
	deviceID := ctx.Param("deviceId")

	device, err := handler.service.GetDevice(ctx.Request.Context(), deviceID)
	if err != nil {
		handler.logger.Error("get device failed", "deviceId", deviceID, "error", err)

		if errors.Is(err, repositories.ErrDeviceNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
			return
		}

		if errors.Is(err, services.ErrDeviceValidation) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch device"})
		return
	}

	ctx.JSON(http.StatusOK, toDeviceResponse(device))
}

func (handler *deviceHandler) listDevices(ctx *gin.Context) {
	page, err := parseQueryInt(ctx.Query("page"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "page must be a valid integer"})
		return
	}

	limit, err := parseQueryInt(ctx.Query("limit"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a valid integer"})
		return
	}

	devices, err := handler.service.ListDevices(ctx.Request.Context(), services.DeviceListParams{
		Page:   page,
		Limit:  limit,
		Status: ctx.Query("status"),
		Search: ctx.Query("search"),
		Tags:   parseTagFilters(ctx),
	})
	if err != nil {
		if errors.Is(err, services.ErrDeviceValidation) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		handler.logger.Error("list devices failed", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch devices"})
		return
	}

	ctx.JSON(http.StatusOK, toDeviceResponses(devices))
}

func toDeviceResponse(device *models.Device) deviceResponse {
	response := deviceResponse{
		ID:              device.ID,
		DeviceID:        device.DeviceID,
		Status:          device.Status,
		FirmwareVersion: device.FirmwareVersion,
		Metadata:        device.Metadata,
		Tags:            device.Tags,
		DesiredState:    device.DesiredState,
		ReportedState:   device.ReportedState,
		CreatedAt:       device.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       device.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if device.LastSeen != nil {
		response.LastSeen = device.LastSeen.Format("2006-01-02T15:04:05Z07:00")
	}

	return response
}

func toDeviceResponses(devices []models.Device) []deviceResponse {
	responses := make([]deviceResponse, len(devices))
	for i, device := range devices {
		deviceCopy := device
		responses[i] = toDeviceResponse(&deviceCopy)
	}
	return responses
}

func parseQueryInt(value string) (int, error) {
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return parsed, nil
}

func parseTagFilters(ctx *gin.Context) []string {
	rawTags := ctx.QueryArray("tags")
	if len(rawTags) == 0 {
		return nil
	}

	parsed := make([]string, 0, len(rawTags))
	for _, rawTagGroup := range rawTags {
		parts := strings.Split(rawTagGroup, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			parsed = append(parsed, trimmed)
		}
	}

	if len(parsed) == 0 {
		return nil
	}

	return parsed
}
