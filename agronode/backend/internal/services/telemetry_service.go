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
	broadcaster TelemetryBroadcaster
}

var ErrValidation = errors.New("telemetry validation failed")

type TelemetryBroadcaster interface {
	Publish(models.TelemetryReading)
}

func NewTelemetryService(repository repositories.TelemetryRepository, logger *slog.Logger) *TelemetryService {
	return &TelemetryService{
		repository: repository,
		logger:     logger,
	}
}

func (service *TelemetryService) SetBroadcaster(broadcaster TelemetryBroadcaster) {
	service.broadcaster = broadcaster
}

func (service *TelemetryService) ProcessTelemetry(context context.Context, reading models.TelemetryReading) error {
	if service.repository == nil {
		return errors.New("telemetry repository is not configured")
	}

	if err := validateTelemetryReading(reading); err != nil {
		return err
	}

	if err := service.repository.Save(context, reading); err != nil {
		return err
	}

	if service.broadcaster != nil {
		service.broadcaster.Publish(reading)
	}

	return nil
}

func (service *TelemetryService) HandleTelemetry(context context.Context, telemetry mqtt.TelemetryEnvelope) error {
	temperature := 0.0
	humidity := 0.0
	if sensorTemperature, hasTemperature := telemetry.Sensors["temperature"]; hasTemperature {
		temperature = sensorTemperature
	}
	if sensorHumidity, hasHumidity := telemetry.Sensors["humidity"]; hasHumidity {
		humidity = sensorHumidity
	}

	sensors := make(map[string]float64, len(telemetry.Sensors))
	for key, value := range telemetry.Sensors {
		sensors[key] = value
	}

	createdAt := time.Now().UTC()
	if telemetry.Timestamp > 0 {
		createdAt = time.Unix(telemetry.Timestamp, 0).UTC()
	}

	reading := models.TelemetryReading{
		DeviceID:    telemetry.DeviceID,
		Temperature: temperature,
		Humidity:    humidity,
		Sensors:     sensors,
		CreatedAt:   createdAt,
	}

	if telemetry.Meta != nil {
		reading.Meta = &models.DeviceMeta{
			Firmware: telemetry.Meta.Firmware,
			IP:       telemetry.Meta.IP,
			RSSI:     telemetry.Meta.RSSI,
			Uptime:   telemetry.Meta.Uptime,
		}
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
		return nil, fmt.Errorf("%w: device id is required", ErrValidation)
	}

	return service.repository.ListByDeviceID(context, deviceID)
}

func (service *TelemetryService) GetTelemetryByDeviceIDWithDateFilter(context context.Context, deviceID string, period string, startDate, endDate *time.Time) ([]models.TelemetryReading, error) {
	if service.repository == nil {
		return nil, errors.New("telemetry repository is not configured")
	}

	if strings.TrimSpace(deviceID) == "" {
		return nil, fmt.Errorf("%w: device id is required", ErrValidation)
	}

	dateRange := calculateDateRange(period, startDate, endDate)
	return service.repository.ListByDeviceIDWithDateRange(context, deviceID, dateRange)
}

func calculateDateRange(period string, startDate, endDate *time.Time) repositories.DateRange {
	now := time.Now().UTC()
	
	switch period {
	case "hour":
		start := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
		end := start.Add(1 * time.Hour)
		return repositories.DateRange{StartDate: &start, EndDate: &end}
	
	case "day":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		end := start.Add(24 * time.Hour)
		return repositories.DateRange{StartDate: &start, EndDate: &end}
	
	case "week":
		// Start of week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, time.UTC)
		end := start.Add(7 * 24 * time.Hour)
		return repositories.DateRange{StartDate: &start, EndDate: &end}
	
	case "month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0)
		return repositories.DateRange{StartDate: &start, EndDate: &end}
	
	case "year":
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(1, 0, 0)
		return repositories.DateRange{StartDate: &start, EndDate: &end}
	
	case "custom":
		return repositories.DateRange{StartDate: startDate, EndDate: endDate}
	
	default:
		// No filter
		return repositories.DateRange{}
	}
}

func (service *TelemetryService) GetLatestTelemetryByDeviceID(context context.Context, deviceID string) (models.TelemetryReading, error) {
	if service.repository == nil {
		return models.TelemetryReading{}, errors.New("telemetry repository is not configured")
	}

	if strings.TrimSpace(deviceID) == "" {
		return models.TelemetryReading{}, fmt.Errorf("%w: device id is required", ErrValidation)
	}

	return service.repository.GetLatestByDeviceID(context, deviceID)
}

func validateTelemetryReading(reading models.TelemetryReading) error {
	if strings.TrimSpace(reading.DeviceID) == "" {
		return fmt.Errorf("%w: device id is required", ErrValidation)
	}

	if reading.CreatedAt.IsZero() {
		return fmt.Errorf("%w: createdAt is required", ErrValidation)
	}

	if len(reading.Sensors) == 0 {
		return fmt.Errorf("%w: at least one sensor value is required", ErrValidation)
	}

	if temperature, hasTemperature := reading.Sensors["temperature"]; hasTemperature {
		if temperature < -100 || temperature > 150 {
			return fmt.Errorf("%w: temperature is out of range", ErrValidation)
		}
	}

	if humidity, hasHumidity := reading.Sensors["humidity"]; hasHumidity {
		if humidity < 0 || humidity > 100 {
			return fmt.Errorf("%w: humidity is out of range", ErrValidation)
		}
	}

	return nil
}
