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
	GetTelemetryByDeviceIDAndSensorID(context.Context, string, string) ([]models.TelemetryReading, error)
	GetTelemetryByDeviceIDWithDateFilter(context.Context, string, string, *time.Time, *time.Time) ([]models.TelemetryReading, error)
	GetLatestTelemetryByDeviceID(context.Context, string) (models.TelemetryReading, error)
}

type telemetryHandler struct {
	logger  *slog.Logger
	service TelemetryQueryService
}

type telemetryResponse struct {
	DeviceID    string             `json:"deviceId"`
	SensorID    string             `json:"sensorId,omitempty"`
	Temperature *float64           `json:"temperature,omitempty"`
	Humidity    *float64           `json:"humidity,omitempty"`
	Sensors     map[string]float64 `json:"sensors,omitempty"`
	CreatedAt   string             `json:"createdAt"`
}

func RegisterTelemetryRoutes(api *gin.RouterGroup, logger *slog.Logger, service TelemetryQueryService) {
	handler := &telemetryHandler{logger: logger, service: service}

	api.GET("/data", handler.getAllData)
	api.GET("/data/:deviceId", handler.getDataByDeviceID)
	api.GET("/data/:deviceId/:sensorId", handler.getDataByDeviceAndSensorID)
	api.GET("/latest/:deviceId", handler.getLatestByDeviceID)
}

func (handler *telemetryHandler) getAllData(context *gin.Context) {
	requestContext, err := requestContextWithOrganizationScope(context)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	readings, err := handler.service.GetAllTelemetry(requestContext)
	if err != nil {
		handler.logger.Error("get all telemetry failed", "error", err)
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch telemetry"})
		return
	}

	context.JSON(http.StatusOK, toTelemetryResponses(readings))
}

func (handler *telemetryHandler) getDataByDeviceID(context *gin.Context) {
	deviceID := context.Param("deviceId")
	requestContext, err := requestContextWithOrganizationScope(context)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Query parameters for date filtering
	period := context.Query("period") // day, week, month, year, custom
	startDateStr := context.Query("startDate")
	endDateStr := context.Query("endDate")

	var readings []models.TelemetryReading
	var serviceErr error

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

		readings, serviceErr = handler.service.GetTelemetryByDeviceIDWithDateFilter(
			requestContext,
			deviceID,
			period,
			startDate,
			endDate,
		)
	} else {
		readings, serviceErr = handler.service.GetTelemetryByDeviceID(requestContext, deviceID)
	}

	if serviceErr != nil {
		handler.logger.Error("get telemetry by device failed", "deviceId", deviceID, "error", serviceErr)
		if errors.Is(serviceErr, services.ErrValidation) {
			context.JSON(http.StatusBadRequest, gin.H{"error": serviceErr.Error()})
			return
		}

		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch telemetry"})
		return
	}

	context.JSON(http.StatusOK, toTelemetryResponses(readings))
}

func (handler *telemetryHandler) getDataByDeviceAndSensorID(context *gin.Context) {
	deviceID := context.Param("deviceId")
	sensorID := context.Param("sensorId")
	requestContext, err := requestContextWithOrganizationScope(context)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	readings, serviceErr := handler.service.GetTelemetryByDeviceIDAndSensorID(requestContext, deviceID, sensorID)
	if serviceErr != nil {
		handler.logger.Error("get telemetry by device and sensor failed", "deviceId", deviceID, "sensorId", sensorID, "error", serviceErr)
		if errors.Is(serviceErr, services.ErrValidation) {
			context.JSON(http.StatusBadRequest, gin.H{"error": serviceErr.Error()})
			return
		}

		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch telemetry"})
		return
	}

	context.JSON(http.StatusOK, toTelemetryResponses(readings))
}

func (handler *telemetryHandler) getLatestByDeviceID(context *gin.Context) {
	deviceID := context.Param("deviceId")
	requestContext, err := requestContextWithOrganizationScope(context)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reading, err := handler.service.GetLatestTelemetryByDeviceID(requestContext, deviceID)
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
	temperature, humidity := resolveTelemetryValues(reading)

	return telemetryResponse{
		DeviceID:    reading.DeviceID,
		SensorID:    reading.SensorID,
		Temperature: temperature,
		Humidity:    humidity,
		Sensors:     reading.Sensors,
		CreatedAt:   createdAt,
	}
}

func resolveTelemetryValues(reading models.TelemetryReading) (*float64, *float64) {
	if value, ok := telemetryValueForKeys(reading, "temperature", "dht11-temp"); ok {
		return &value, telemetryHumidityValue(reading)
	}

	return nil, telemetryHumidityValue(reading)
}

func telemetryHumidityValue(reading models.TelemetryReading) *float64 {
	if value, ok := telemetryValueForKeys(reading, "humidity", "humidity_dht11", "dht11-humidity"); ok {
		return &value
	}

	return nil
}

func telemetryValueForKeys(reading models.TelemetryReading, keys ...string) (float64, bool) {
	if len(reading.Sensors) > 0 {
		for _, key := range keys {
			if value, ok := reading.Sensors[key]; ok {
				return value, true
			}
		}
	}

	for _, key := range keys {
		if reading.SensorID != key {
			continue
		}

		switch key {
		case "temperature", "dht11-temp":
			return reading.Temperature, true
		case "humidity", "humidity_dht11", "dht11-humidity":
			return reading.Humidity, true
		}
	}

	if reading.SensorID == "" {
		for _, key := range keys {
			switch key {
			case "temperature":
				return reading.Temperature, true
			case "humidity":
				return reading.Humidity, true
			}
		}
	}

	return 0, false
}
