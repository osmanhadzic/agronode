package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"agronode/backend/internal/models"
	"agronode/backend/internal/repositories"
	"agronode/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type TelemetryQueryService interface {
	GetAllTelemetry(context.Context) ([]models.TelemetryReading, error)
	GetTelemetryByDeviceID(context.Context, string) ([]models.TelemetryReading, error)
	GetTelemetryByDeviceIDWithDateFilter(context.Context, string, string, *time.Time, *time.Time) ([]models.TelemetryReading, error)
	GetLatestTelemetryByDeviceID(context.Context, string) (models.TelemetryReading, error)
}

type telemetryHandler struct {
	logger  *slog.Logger
	service TelemetryQueryService
}

type telemetryResponse struct {
	DeviceID    string  `json:"deviceId"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	Sensors     map[string]float64 `json:"sensors,omitempty"`
	CreatedAt   string  `json:"createdAt"`
}

func RegisterTelemetryRoutes(api *gin.RouterGroup, logger *slog.Logger, service TelemetryQueryService) {
	handler := &telemetryHandler{logger: logger, service: service}

	api.GET("/data", handler.getAllData)
	api.GET("/data/:deviceId", handler.getDataByDeviceID)
	api.GET("/latest/:deviceId", handler.getLatestByDeviceID)
}

func (handler *telemetryHandler) getAllData(context *gin.Context) {
	readings, err := handler.service.GetAllTelemetry(context.Request.Context())
	if err != nil {
		handler.logger.Error("get all telemetry failed", "error", err)
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch telemetry"})
		return
	}

	context.JSON(http.StatusOK, toTelemetryResponses(readings))
}

func (handler *telemetryHandler) getDataByDeviceID(context *gin.Context) {
	deviceID := context.Param("deviceId")
	
	// Query parameters for date filtering
	period := context.Query("period") // day, week, month, year, custom
	startDateStr := context.Query("startDate")
	endDateStr := context.Query("endDate")

	var readings []models.TelemetryReading
	var err error

	// If period or date range is specified, use filtered query
	if period != "" || startDateStr != "" || endDateStr != "" {
		var startDate, endDate *time.Time
		
		if startDateStr != "" {
			parsed, parseErr := time.Parse(time.RFC3339, startDateStr)
			if parseErr != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "invalid startDate format, use RFC3339"})
				return
			}
			startDate = &parsed
		}
		
		if endDateStr != "" {
			parsed, parseErr := time.Parse(time.RFC3339, endDateStr)
			if parseErr != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "invalid endDate format, use RFC3339"})
				return
			}
			endDate = &parsed
		}
		
		readings, err = handler.service.GetTelemetryByDeviceIDWithDateFilter(
			context.Request.Context(), 
			deviceID, 
			period, 
			startDate, 
			endDate,
		)
	} else {
		readings, err = handler.service.GetTelemetryByDeviceID(context.Request.Context(), deviceID)
	}

	if err != nil {
		handler.logger.Error("get telemetry by device failed", "deviceId", deviceID, "error", err)
		if errors.Is(err, services.ErrValidation) {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch telemetry"})
		return
	}

	context.JSON(http.StatusOK, toTelemetryResponses(readings))
}

func (handler *telemetryHandler) getLatestByDeviceID(context *gin.Context) {
	deviceID := context.Param("deviceId")

	reading, err := handler.service.GetLatestTelemetryByDeviceID(context.Request.Context(), deviceID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			context.JSON(http.StatusNotFound, gin.H{"error": "telemetry not found"})
			return
		}

		if errors.Is(err, services.ErrValidation) {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		handler.logger.Error("get latest telemetry failed", "deviceId", deviceID, "error", err)
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch telemetry"})
		return
	}

	context.JSON(http.StatusOK, toTelemetryResponse(reading))
}

func toTelemetryResponses(readings []models.TelemetryReading) []telemetryResponse {
	responses := make([]telemetryResponse, 0, len(readings))
	for _, reading := range readings {
		responses = append(responses, toTelemetryResponse(reading))
	}

	return responses
}

func toTelemetryResponse(reading models.TelemetryReading) telemetryResponse {
	createdAt := reading.CreatedAt.UTC().Format(time.RFC3339)

	return telemetryResponse{
		DeviceID:    reading.DeviceID,
		Temperature: reading.Temperature,
		Humidity:    reading.Humidity,
		Sensors:     reading.Sensors,
		CreatedAt:   createdAt,
	}
}
