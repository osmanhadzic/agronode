package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"agronode/backend/internal/models"
	"agronode/backend/internal/mqtt"
	"agronode/backend/internal/repositories"
)

type telemetryRepositoryStub struct {
	saved     []models.TelemetryReading
	saveError error
}

type devicePresenceUpdaterStub struct {
	updatedDeviceID string
	updatedSeenAt   time.Time
	err             error
}

func (stub *devicePresenceUpdaterStub) UpdatePresence(_ context.Context, deviceID string, seenAt time.Time) error {
	stub.updatedDeviceID = deviceID
	stub.updatedSeenAt = seenAt
	return stub.err
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
			Sensors: map[string]float64{
				"temperature": 24.5,
				"humidity":    60,
			},
			CreatedAt: now,
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

func TestTelemetryService_HandleTelemetry_Presence(t *testing.T) {
	t.Run("updates device presence after telemetry processing", func(t *testing.T) {
		repository := &telemetryRepositoryStub{}
		presenceUpdater := &devicePresenceUpdaterStub{}
		service := NewTelemetryService(repository, nil)
		service.SetPresenceUpdater(presenceUpdater)

		envelope := mqtt.TelemetryEnvelope{
			DeviceID:  "esp32-lab",
			Timestamp: time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC).Unix(),
			Sensors: map[string]float64{
				"temperature": 24.5,
				"humidity":    61,
			},
		}

		err := service.HandleTelemetry(context.Background(), envelope)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if presenceUpdater.updatedDeviceID != "esp32-lab" {
			t.Fatalf("expected updated device id %q, got %q", "esp32-lab", presenceUpdater.updatedDeviceID)
		}

		if presenceUpdater.updatedSeenAt.IsZero() {
			t.Fatal("expected non-zero presence updated timestamp")
		}
	})

	t.Run("does not fail telemetry processing when device not found", func(t *testing.T) {
		repository := &telemetryRepositoryStub{}
		presenceUpdater := &devicePresenceUpdaterStub{err: repositories.ErrDeviceNotFound}
		service := NewTelemetryService(repository, nil)
		service.SetPresenceUpdater(presenceUpdater)

		envelope := mqtt.TelemetryEnvelope{
			DeviceID: "esp32-lab",
			Sensors: map[string]float64{
				"temperature": 23,
			},
		}

		err := service.HandleTelemetry(context.Background(), envelope)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(repository.saved) != 1 {
			t.Fatalf("expected telemetry to be saved once, got %d", len(repository.saved))
		}
	})
}

func TestTelemetryService_HandleTelemetry_Discovery(t *testing.T) {
	t.Run("forwards discovered sensor names", func(t *testing.T) {
		repository := &telemetryRepositoryStub{}
		presenceUpdater := &devicePresenceUpdaterStub{}
		discoveryUpdater := &deviceSensorDiscoveryUpdaterStub{}
		service := NewTelemetryService(repository, nil)
		service.SetPresenceUpdater(presenceUpdater)
		service.SetSensorDiscoveryUpdater(discoveryUpdater)

		envelope := mqtt.TelemetryEnvelope{
			DeviceID: "esp32-lab",
			Sensors: map[string]float64{
				"temperature":   24.5,
				"humidity":      61,
				"co2":           410,
				"soil_moisture": 35,
			},
		}

		err := service.HandleTelemetry(context.Background(), envelope)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if discoveryUpdater.updatedDeviceID != "esp32-lab" {
			t.Fatalf("expected updated device id %q, got %q", "esp32-lab", discoveryUpdater.updatedDeviceID)
		}

		if len(discoveryUpdater.updatedSensors) != 4 {
			t.Fatalf("expected 4 discovered sensors, got %v", discoveryUpdater.updatedSensors)
		}
	})
}
