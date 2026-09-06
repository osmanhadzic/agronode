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

type triggerPublisherStub struct {
	commands []mqtt.ActivationCommand
	err      error
}

func (publisher *triggerPublisherStub) PublishActivationCommand(_ context.Context, command mqtt.ActivationCommand) error {
	if publisher.err != nil {
		return publisher.err
	}

	publisher.commands = append(publisher.commands, command)
	return nil
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
			Sensors: map[string]float64{
				"temperature": 24.5,
			},
			CreatedAt: time.Now().UTC(),
		})

		if !errors.Is(err, ErrValidation) {
			t.Fatalf("expected ErrValidation, got %v", err)
		}

		if len(repository.saved) != 0 {
			t.Fatalf("expected no saved readings, got %d", len(repository.saved))
		}
	})
}

func TestTelemetryService_SetSensorTrigger(t *testing.T) {
	t.Run("rejects invalid min max combination", func(t *testing.T) {
		repository := &telemetryRepositoryStub{}
		service := NewTelemetryService(repository, nil)

		minValue := 30.0
		maxValue := 20.0
		err := service.SetSensorTrigger(context.Background(), "esp32-lab", "temperature", models.SensorTrigger{
			Min: &minValue,
			Max: &maxValue,
		})

		if !errors.Is(err, ErrValidation) {
			t.Fatalf("expected ErrValidation, got %v", err)
		}
	})

	t.Run("stores and returns trigger", func(t *testing.T) {
		repository := &telemetryRepositoryStub{}
		service := NewTelemetryService(repository, nil)

		minValue := 10.0
		maxValue := 50.0
		err := service.SetSensorTrigger(context.Background(), "esp32-lab", "humidity", models.SensorTrigger{
			Min: &minValue,
			Max: &maxValue,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		trigger, err := service.GetSensorTrigger(context.Background(), "esp32-lab", "humidity")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if trigger.Min == nil || *trigger.Min != minValue {
			t.Fatalf("expected min %v, got %v", minValue, trigger.Min)
		}

		if trigger.Max == nil || *trigger.Max != maxValue {
			t.Fatalf("expected max %v, got %v", maxValue, trigger.Max)
		}
	})

	t.Run("returns not found for missing sensor on existing device", func(t *testing.T) {
		repository := &telemetryRepositoryStub{}
		service := NewTelemetryService(repository, nil)

		maxValue := 70.0
		err := service.SetSensorTrigger(context.Background(), "esp32-lab", "humidity", models.SensorTrigger{
			Max: &maxValue,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		_, err = service.GetSensorTrigger(context.Background(), "esp32-lab", "temperature")
		if !errors.Is(err, repositories.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestTelemetryService_GenericTriggerActivation(t *testing.T) {
	repository := &telemetryRepositoryStub{}
	publisher := &triggerPublisherStub{}
	service := NewTelemetryService(repository, nil)
	service.SetTriggerPublisher(publisher)

	maxValue := 700.0
	err := service.SetSensorTrigger(context.Background(), "esp32-lab", "co2", models.SensorTrigger{Max: &maxValue})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	now := time.Now().UTC()
	firstReading := models.TelemetryReading{
		DeviceID: "esp32-lab",
		Sensors: map[string]float64{
			"co2":         800,
			"temperature": 24,
		},
		CreatedAt: now,
	}

	if err := service.ProcessTelemetry(context.Background(), firstReading); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(publisher.commands) != 1 {
		t.Fatalf("expected 1 activation command, got %d", len(publisher.commands))
	}

	command := publisher.commands[0]
	if command.Sensor != "co2" || command.Trigger != "above_max" || command.LimitType != "max" {
		t.Fatalf("unexpected command: %+v", command)
	}

	secondReading := models.TelemetryReading{
		DeviceID: "esp32-lab",
		Sensors: map[string]float64{
			"co2":         810,
			"temperature": 24,
		},
		CreatedAt: now.Add(1 * time.Second),
	}

	if err := service.ProcessTelemetry(context.Background(), secondReading); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(publisher.commands) != 1 {
		t.Fatalf("expected still 1 activation command while above max, got %d", len(publisher.commands))
	}
}
