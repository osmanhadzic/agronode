package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"agronode/backend/internal/models"
	"agronode/backend/internal/mqtt"
	"agronode/backend/internal/repositories"
)

type TelemetryService struct {
	repository repositories.TelemetryRepository
	logger     *slog.Logger
}

func NewTelemetryService(repository repositories.TelemetryRepository, logger *slog.Logger) *TelemetryService {
	return &TelemetryService{
		repository: repository,
		logger:     logger,
	}
}

func (service *TelemetryService) ProcessTelemetry(context context.Context, reading models.TelemetryReading) error {
	if service.repository == nil {
		return errors.New("telemetry repository is not configured")
	}

	if err := validateTelemetryReading(reading); err != nil {
		return err
	}

	return service.repository.Save(context, reading)
}

func (service *TelemetryService) HandleTelemetry(context context.Context, telemetry mqtt.TelemetryEnvelope) error {
	temperature, hasTemperature := telemetry.Sensors["temperature"]
	humidity, hasHumidity := telemetry.Sensors["humidity"]
	if !hasTemperature || !hasHumidity {
		return fmt.Errorf("required sensors missing: temperature and humidity are required")
	}

	createdAt := time.Now().UTC()
	if telemetry.Timestamp > 0 {
		createdAt = time.Unix(telemetry.Timestamp, 0).UTC()
	}

	reading := models.TelemetryReading{
		DeviceID:    telemetry.DeviceID,
		Temperature: temperature,
		Humidity:    humidity,
		CreatedAt:   createdAt,
	}

	if err := service.ProcessTelemetry(context, reading); err != nil {
		return err
	}

	if service.logger != nil {
		service.logger.Info("telemetry processed", "deviceId", reading.DeviceID, "createdAt", reading.CreatedAt)
	}

	return nil
}

func (service *TelemetryService) GetAllTelemetry(context context.Context) ([]models.TelemetryReading, error) {
	if service.repository == nil {
		return nil, errors.New("telemetry repository is not configured")
	}

	return service.repository.List(context)
}

func (service *TelemetryService) GetTelemetryByDeviceID(context context.Context, deviceID string) ([]models.TelemetryReading, error) {
	if service.repository == nil {
		return nil, errors.New("telemetry repository is not configured")
	}

	if strings.TrimSpace(deviceID) == "" {
		return nil, errors.New("device id is required")
	}

	return service.repository.ListByDeviceID(context, deviceID)
}

func (service *TelemetryService) GetLatestTelemetryByDeviceID(context context.Context, deviceID string) (models.TelemetryReading, error) {
	if service.repository == nil {
		return models.TelemetryReading{}, errors.New("telemetry repository is not configured")
	}

	if strings.TrimSpace(deviceID) == "" {
		return models.TelemetryReading{}, errors.New("device id is required")
	}

	return service.repository.GetLatestByDeviceID(context, deviceID)
}

func validateTelemetryReading(reading models.TelemetryReading) error {
	if strings.TrimSpace(reading.DeviceID) == "" {
		return errors.New("device id is required")
	}

	if reading.CreatedAt.IsZero() {
		return errors.New("createdAt is required")
	}

	if reading.Temperature < -100 || reading.Temperature > 150 {
		return fmt.Errorf("temperature is out of range")
	}

	if reading.Humidity < 0 || reading.Humidity > 100 {
		return fmt.Errorf("humidity is out of range")
	}

	return nil
}
