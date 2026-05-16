package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"agronode/backend/internal/models"
)

type telemetryRepositoryStub struct {
	saved     []models.TelemetryReading
	saveError error
}

func (repository *telemetryRepositoryStub) Save(_ context.Context, reading models.TelemetryReading) error {
	if repository.saveError != nil {
		return repository.saveError
	}
	repository.saved = append(repository.saved, reading)
	return nil
}

func (repository *telemetryRepositoryStub) List(context.Context) ([]models.TelemetryReading, error) {
	return nil, nil
}

func (repository *telemetryRepositoryStub) ListByDeviceID(context.Context, string) ([]models.TelemetryReading, error) {
	return nil, nil
}

func (repository *telemetryRepositoryStub) GetLatestByDeviceID(context.Context, string) (models.TelemetryReading, error) {
	return models.TelemetryReading{}, nil
}

func TestTelemetryService_ProcessTelemetry(t *testing.T) {
	t.Run("saves valid telemetry", func(t *testing.T) {
		repository := &telemetryRepositoryStub{}
		service := NewTelemetryService(repository, nil)

		now := time.Now().UTC().Truncate(time.Second)
		reading := models.TelemetryReading{
			DeviceID:    "esp32-lab",
			Temperature: 24.5,
			Humidity:    60,
			CreatedAt:   now,
		}

		err := service.ProcessTelemetry(context.Background(), reading)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(repository.saved) != 1 {
			t.Fatalf("expected 1 saved reading, got %d", len(repository.saved))
		}

		saved := repository.saved[0]
		if saved.DeviceID != reading.DeviceID {
			t.Fatalf("expected deviceID %q, got %q", reading.DeviceID, saved.DeviceID)
		}
	})

	t.Run("rejects invalid telemetry", func(t *testing.T) {
		repository := &telemetryRepositoryStub{}
		service := NewTelemetryService(repository, nil)

		err := service.ProcessTelemetry(context.Background(), models.TelemetryReading{
			DeviceID:    "   ",
			Temperature: 24.5,
			Humidity:    60,
			CreatedAt:   time.Now().UTC(),
		})

		if !errors.Is(err, ErrValidation) {
			t.Fatalf("expected ErrValidation, got %v", err)
		}

		if len(repository.saved) != 0 {
			t.Fatalf("expected no saved readings, got %d", len(repository.saved))
		}
	})
}
