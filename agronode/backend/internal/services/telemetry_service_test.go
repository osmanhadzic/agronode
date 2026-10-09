package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agronode/backend/internal/fuzzy"
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

type deviceMetadataUpdaterStub struct {
	updatedDeviceID string
	updatedMeta     *models.DeviceMeta
	err             error
}

type sensorRepositoryStub struct {
	upserts []struct {
		deviceID string
		sensorID string
	}
	listResult []models.Sensor
	listError  error
	upsertError error
}

func (stub *devicePresenceUpdaterStub) UpdatePresence(_ context.Context, deviceID string, seenAt time.Time) error {
	stub.updatedDeviceID = deviceID
	stub.updatedSeenAt = seenAt
	return stub.err
}

func (stub *deviceMetadataUpdaterStub) UpdateMetadataFromTelemetry(_ context.Context, deviceID string, meta *models.DeviceMeta) error {
	stub.updatedDeviceID = deviceID
	stub.updatedMeta = meta
	return stub.err
}

func (stub *sensorRepositoryStub) Upsert(_ context.Context, deviceID, sensorID string) error {
	if stub.upsertError != nil {
		return stub.upsertError
	}

	stub.upserts = append(stub.upserts, struct {
		deviceID string
		sensorID string
	}{deviceID: deviceID, sensorID: sensorID})
	return nil
}

func (stub *sensorRepositoryStub) ListByDeviceID(_ context.Context, _ string) ([]models.Sensor, error) {
	return stub.listResult, stub.listError
}

type triggerPublisherStub struct {
	commands []mqtt.ActivationCommand
	err      error
}

type triggerRepositoryStub struct {
	executions       []models.IrrigationExecution
	saveExecutionErr error
}

func (stub *triggerRepositoryStub) Upsert(context.Context, string, string, models.SensorTrigger) error {
	return nil
}

func (stub *triggerRepositoryStub) GetByDeviceAndSensor(context.Context, string, string) (models.SensorTrigger, error) {
	return models.SensorTrigger{}, repositories.ErrNotFound
}

func (stub *triggerRepositoryStub) ListByDeviceID(context.Context, string) (map[string]models.SensorTrigger, error) {
	return map[string]models.SensorTrigger{}, nil
}

func (stub *triggerRepositoryStub) DeleteByDeviceAndSensor(context.Context, string, string) error {
	return nil
}

func (stub *triggerRepositoryStub) SaveExecution(_ context.Context, execution models.IrrigationExecution) error {
	if stub.saveExecutionErr != nil {
		return stub.saveExecutionErr
	}

	stub.executions = append(stub.executions, execution)
	return nil
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

func (repository *telemetryRepositoryStub) ListByDeviceIDAndSensorID(context.Context, string, string) ([]models.TelemetryReading, error) {
	return nil, nil
}

func (repository *telemetryRepositoryStub) ListByDeviceIDWithDateRange(_ context.Context, _ string, _ repositories.DateRange) ([]models.TelemetryReading, error) {
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
			SensorID:    "dht11",
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

	t.Run("upserts sensor for explicit sensor id", func(t *testing.T) {
		repository := &telemetryRepositoryStub{}
		sensorRepository := &sensorRepositoryStub{}
		service := NewTelemetryService(repository, nil)
		service.SetSensorRepository(sensorRepository)

		err := service.ProcessTelemetry(context.Background(), models.TelemetryReading{
			DeviceID:    "esp32-lab",
			SensorID:    "dht11",
			Temperature: 24.5,
			Humidity:    60,
			Sensors: map[string]float64{
				"temperature": 24.5,
				"humidity":    60,
			},
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(sensorRepository.upserts) != 1 {
			t.Fatalf("expected 1 sensor upsert, got %d", len(sensorRepository.upserts))
		}

		if sensorRepository.upserts[0].sensorID != "dht11" {
			t.Fatalf("expected upsert sensor id %q, got %q", "dht11", sensorRepository.upserts[0].sensorID)
		}
	})

	t.Run("upserts single discovered sensor when sensor id is missing", func(t *testing.T) {
		repository := &telemetryRepositoryStub{}
		sensorRepository := &sensorRepositoryStub{}
		service := NewTelemetryService(repository, nil)
		service.SetSensorRepository(sensorRepository)

		err := service.ProcessTelemetry(context.Background(), models.TelemetryReading{
			DeviceID: "esp32-lab",
			Sensors: map[string]float64{
				"co2": 420,
			},
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(sensorRepository.upserts) != 1 {
			t.Fatalf("expected 1 sensor upsert, got %d", len(sensorRepository.upserts))
		}

		if sensorRepository.upserts[0].sensorID != "co2" {
			t.Fatalf("expected upsert sensor id %q, got %q", "co2", sensorRepository.upserts[0].sensorID)
		}
	})

	t.Run("returns error when sensor upsert fails", func(t *testing.T) {
		repository := &telemetryRepositoryStub{}
		sensorRepository := &sensorRepositoryStub{upsertError: errors.New("upsert failed")}
		service := NewTelemetryService(repository, nil)
		service.SetSensorRepository(sensorRepository)

		err := service.ProcessTelemetry(context.Background(), models.TelemetryReading{
			DeviceID: "esp32-lab",
			SensorID: "dht11",
			Sensors: map[string]float64{
				"temperature": 24.5,
			},
			CreatedAt: time.Now().UTC(),
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if len(repository.saved) != 0 {
			t.Fatalf("expected no telemetry save on sensor upsert failure, got %d", len(repository.saved))
		}
	})
}

func TestTelemetryService_GetTelemetryByDeviceIDAndSensorID(t *testing.T) {
	repository := &telemetryRepositoryStub{}
	service := NewTelemetryService(repository, nil)

	_, err := service.GetTelemetryByDeviceIDAndSensorID(context.Background(), "esp32-lab", "temperature")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestTelemetryService_HandleTelemetry_Presence(t *testing.T) {
	t.Run("updates device presence after telemetry processing", func(t *testing.T) {
		repository := &telemetryRepositoryStub{}
		presenceUpdater := &devicePresenceUpdaterStub{}
		service := NewTelemetryService(repository, nil)
		service.SetPresenceUpdater(presenceUpdater)

		envelope := mqtt.TelemetryEnvelope{
			DeviceID:  "esp32-lab",
			SensorID:  "dht11",
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
			SensorID: "temperature",
			Sensors: map[string]float64{
				"temperature": 23,
			},
		}

		err := service.HandleTelemetry(context.Background(), envelope)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("updates device metadata when meta payload exists", func(t *testing.T) {
		repository := &telemetryRepositoryStub{}
		metadataUpdater := &deviceMetadataUpdaterStub{}
		service := NewTelemetryService(repository, nil)
		service.SetMetadataUpdater(metadataUpdater)

		envelope := mqtt.TelemetryEnvelope{
			DeviceID: "esp32-lab",
			SensorID: "temperature",
			Meta: &mqtt.DeviceMeta{
				Firmware: "1.0.1",
				IP:       "192.168.1.50",
				RSSI:     -65,
				Uptime:   123,
			},
			Sensors: map[string]float64{
				"temperature": 23,
			},
		}

		err := service.HandleTelemetry(context.Background(), envelope)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if metadataUpdater.updatedDeviceID != "esp32-lab" {
			t.Fatalf("expected updated device id %q, got %q", "esp32-lab", metadataUpdater.updatedDeviceID)
		}

		if metadataUpdater.updatedMeta == nil {
			t.Fatal("expected metadata updater to receive telemetry meta")
		}

		if metadataUpdater.updatedMeta.RSSI != -65 {
			t.Fatalf("expected rssi -65, got %d", metadataUpdater.updatedMeta.RSSI)
		}
	})

	t.Run("updates device metadata from signal_strength sensor when meta is missing", func(t *testing.T) {
		repository := &telemetryRepositoryStub{}
		metadataUpdater := &deviceMetadataUpdaterStub{}
		service := NewTelemetryService(repository, nil)
		service.SetMetadataUpdater(metadataUpdater)

		envelope := mqtt.TelemetryEnvelope{
			DeviceID: "esp32-lab",
			SensorID: "temperature",
			Sensors: map[string]float64{
				"temperature":     23,
				"signal_strength": -71,
			},
		}

		err := service.HandleTelemetry(context.Background(), envelope)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if metadataUpdater.updatedMeta == nil {
			t.Fatal("expected metadata updater to receive fallback meta")
		}

		if metadataUpdater.updatedMeta.RSSI != -71 {
			t.Fatalf("expected fallback rssi -71, got %d", metadataUpdater.updatedMeta.RSSI)
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

	t.Run("persists sensor entry when trigger is configured", func(t *testing.T) {
		repository := &telemetryRepositoryStub{}
		sensorRepository := &sensorRepositoryStub{}
		service := NewTelemetryService(repository, nil)
		service.SetSensorRepository(sensorRepository)

		maxValue := 50.0
		err := service.SetSensorTrigger(context.Background(), "esp32-lab", "temperature", models.SensorTrigger{Max: &maxValue})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(sensorRepository.upserts) != 1 {
			t.Fatalf("expected 1 sensor upsert, got %d", len(sensorRepository.upserts))
		}

		if sensorRepository.upserts[0].deviceID != "esp32-lab" || sensorRepository.upserts[0].sensorID != "temperature" {
			t.Fatalf("unexpected sensor upsert: %#v", sensorRepository.upserts[0])
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

		if len(repository.saved) != 0 {
			t.Fatalf("expected no telemetry rows to be saved when storing a trigger, got %d", len(repository.saved))
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

		if trigger.TargetDeviceID != "esp32-lab" {
			t.Fatalf("expected target device id %q, got %q", "esp32-lab", trigger.TargetDeviceID)
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

func TestTelemetryService_SetSensorTrigger_FuzzyValidation(t *testing.T) {
	repository := &telemetryRepositoryStub{}
	service := NewTelemetryService(repository, nil)

	// invalid fuzzy JSON
	invalidJSON := []byte(`{"membershipFunctions": [ { "name": "a" } ] }`)
	err := service.SetSensorTrigger(context.Background(), "esp32-lab", "temp", models.SensorTrigger{FuzzyConfig: invalidJSON})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation for invalid fuzzy json, got %v", err)
	}

	// valid fuzzy but semantic error (triangle wrong params)
	badTrig := fuzzy.FuzzyTrigger{MembershipFunctions: []fuzzy.MembershipFunction{{Name: "m", Sensor: "temp", Type: "triangle", Parameters: []float64{10, 5, 0}}}, Rules: []fuzzy.Rule{{Name: "r", Conditions: []fuzzy.Condition{{Sensor: "temp", Membership: "m"}}, Action: fuzzy.Action{Type: "x", Value: 1}}}}
	data, _ := json.Marshal(badTrig)
	err = service.SetSensorTrigger(context.Background(), "esp32-lab", "temp", models.SensorTrigger{FuzzyConfig: data})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation for semantic fuzzy error, got %v", err)
	}

	// valid fuzzy config should pass
	goodTrig := fuzzy.FuzzyTrigger{MembershipFunctions: []fuzzy.MembershipFunction{{Name: "m", Sensor: "temp", Type: "triangle", Parameters: []float64{0, 5, 10}}}, Rules: []fuzzy.Rule{{Name: "r", Conditions: []fuzzy.Condition{{Sensor: "temp", Membership: "m"}}, Action: fuzzy.Action{Type: "x", Value: 1}}}}
	data, _ = json.Marshal(goodTrig)
	err = service.SetSensorTrigger(context.Background(), "esp32-lab", "temp", models.SensorTrigger{FuzzyConfig: data})
	if err != nil {
		t.Fatalf("expected no error for valid fuzzy, got %v", err)
	}
}

func TestTelemetryService_GenericTriggerActivation_TargetDevice(t *testing.T) {
	repository := &telemetryRepositoryStub{}
	publisher := &triggerPublisherStub{}
	service := NewTelemetryService(repository, nil)
	service.SetTriggerPublisher(publisher)

	maxValue := 700.0
	err := service.SetSensorTrigger(context.Background(), "esp32-lab", "co2", models.SensorTrigger{
		Max:            &maxValue,
		TargetDeviceID: "pump-node-1",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	now := time.Now().UTC()
	firstReading := models.TelemetryReading{
		DeviceID: "esp32-lab",
		SensorID: "co2",
		Sensors: map[string]float64{
			"co2": 800,
		},
		CreatedAt: now,
	}

	if err := service.ProcessTelemetry(context.Background(), firstReading); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(publisher.commands) != 1 {
		t.Fatalf("expected 1 activation command, got %d", len(publisher.commands))
	}

	if publisher.commands[0].DeviceID != "pump-node-1" {
		t.Fatalf("expected activation target device %q, got %q", "pump-node-1", publisher.commands[0].DeviceID)
	}
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
			SensorID: "temperature",
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
		SensorID: "co2",
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
		SensorID: "co2",
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

func TestTelemetryService_PersistsExecutionAttempts(t *testing.T) {
	t.Run("stores succeeded execution status", func(t *testing.T) {
		repository := &telemetryRepositoryStub{}
		publisher := &triggerPublisherStub{}
		triggerRepository := &triggerRepositoryStub{}
		service := NewTelemetryService(repository, nil)
		service.SetTriggerPublisher(publisher)
		service.SetTriggerRepository(triggerRepository)

		maxValue := 700.0
		err := service.SetSensorTrigger(context.Background(), "esp32-lab", "co2", models.SensorTrigger{Max: &maxValue})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		reading := models.TelemetryReading{
			DeviceID: "esp32-lab",
			SensorID: "co2",
			Sensors: map[string]float64{
				"co2": 800,
			},
			CreatedAt: time.Now().UTC(),
		}

		err = service.ProcessTelemetry(context.Background(), reading)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(triggerRepository.executions) != 1 {
			t.Fatalf("expected 1 persisted execution, got %d", len(triggerRepository.executions))
		}

		execution := triggerRepository.executions[0]
		if execution.Status != models.IrrigationExecutionStatusSucceeded {
			t.Fatalf("expected status %q, got %q", models.IrrigationExecutionStatusSucceeded, execution.Status)
		}

		if execution.RuleID != "above_max" {
			t.Fatalf("expected rule id %q, got %q", "above_max", execution.RuleID)
		}

		if execution.ErrorMessage != nil {
			t.Fatalf("expected nil error message for succeeded execution, got %q", *execution.ErrorMessage)
		}
	})

	t.Run("stores failed execution error details", func(t *testing.T) {
		repository := &telemetryRepositoryStub{}
		publisher := &triggerPublisherStub{err: errors.New("broker unavailable")}
		triggerRepository := &triggerRepositoryStub{}
		service := NewTelemetryService(repository, nil)
		service.SetTriggerPublisher(publisher)
		service.SetTriggerRepository(triggerRepository)

		maxValue := 700.0
		err := service.SetSensorTrigger(context.Background(), "esp32-lab", "co2", models.SensorTrigger{Max: &maxValue})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		reading := models.TelemetryReading{
			DeviceID: "esp32-lab",
			SensorID: "co2",
			Sensors: map[string]float64{
				"co2": 800,
			},
			CreatedAt: time.Now().UTC(),
		}

		err = service.ProcessTelemetry(context.Background(), reading)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(triggerRepository.executions) != 1 {
			t.Fatalf("expected 1 persisted execution, got %d", len(triggerRepository.executions))
		}

		execution := triggerRepository.executions[0]
		if execution.Status != models.IrrigationExecutionStatusFailed {
			t.Fatalf("expected status %q, got %q", models.IrrigationExecutionStatusFailed, execution.Status)
		}

		if execution.ErrorMessage == nil {
			t.Fatalf("expected execution error details to be persisted")
		}

		if *execution.ErrorMessage != "broker unavailable" {
			t.Fatalf("expected error message %q, got %q", "broker unavailable", *execution.ErrorMessage)
		}
	})
}

func TestTelemetryService_FuzzyHTTPAction(t *testing.T) {
	repository := &telemetryRepositoryStub{}
	service := NewTelemetryService(repository, nil)

	received := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received++
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	service.SetHTTPClient(server.Client())

	fuzzyTrigger := fuzzy.FuzzyTrigger{
		MembershipFunctions: []fuzzy.MembershipFunction{
			{Name: "hot", Sensor: "temperature", Type: "triangle", Parameters: []float64{20, 30, 40}},
		},
		Rules: []fuzzy.Rule{
			{
				Name:       "send_webhook",
				Conditions: []fuzzy.Condition{{Sensor: "temperature", Membership: "hot"}},
				Operator:   "AND",
				Action: fuzzy.Action{
					Type:   "http",
					Value:  1,
					URL:    server.URL,
					Method: "POST",
				},
			},
		},
	}

	data, err := json.Marshal(fuzzyTrigger)
	if err != nil {
		t.Fatalf("expected marshaled fuzzy trigger, got %v", err)
	}

	err = service.SetSensorTrigger(context.Background(), "esp32-lab", "temperature", models.SensorTrigger{FuzzyConfig: data})
	if err != nil {
		t.Fatalf("expected no error setting fuzzy trigger, got %v", err)
	}

	err = service.ProcessTelemetry(context.Background(), models.TelemetryReading{
		DeviceID: "esp32-lab",
		SensorID: "temperature",
		Sensors: map[string]float64{
			"temperature": 35,
		},
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("expected no error processing telemetry, got %v", err)
	}

	if received != 1 {
		t.Fatalf("expected 1 outbound webhook request, got %d", received)
	}
}
