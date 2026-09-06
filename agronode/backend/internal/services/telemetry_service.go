package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"agronode/backend/internal/models"
	"agronode/backend/internal/mqtt"
	"agronode/backend/internal/repositories"
)

type TelemetryService struct {
	repository        repositories.TelemetryRepository
	triggerRepository repositories.TriggerRepository
	logger            *slog.Logger
	broadcaster       TelemetryBroadcaster
	triggerPublisher  TriggerCommandPublisher
	triggers          map[string]map[string]models.SensorTrigger
	triggerState      map[string]map[string]sensorTriggerState
	triggerMutex      sync.RWMutex
}

var ErrValidation = errors.New("telemetry validation failed")

type TelemetryBroadcaster interface {
	Publish(models.TelemetryReading)
}

type TriggerCommandPublisher interface {
	PublishActivationCommand(context.Context, mqtt.ActivationCommand) error
}

type sensorTriggerState struct {
	MinActive bool
	MaxActive bool
}

func NewTelemetryService(repository repositories.TelemetryRepository, logger *slog.Logger) *TelemetryService {
	return &TelemetryService{
		repository:   repository,
		logger:       logger,
		triggers:     make(map[string]map[string]models.SensorTrigger),
		triggerState: make(map[string]map[string]sensorTriggerState),
	}
}

func (service *TelemetryService) SetBroadcaster(broadcaster TelemetryBroadcaster) {
	service.broadcaster = broadcaster
}

func (service *TelemetryService) SetTriggerPublisher(publisher TriggerCommandPublisher) {
	service.triggerPublisher = publisher
}

func (service *TelemetryService) SetTriggerRepository(repository repositories.TriggerRepository) {
	service.triggerRepository = repository
}

func (service *TelemetryService) SetSensorTrigger(_ context.Context, deviceID, sensor string, trigger models.SensorTrigger) error {
	trimmedDeviceID := strings.TrimSpace(deviceID)
	if trimmedDeviceID == "" {
		return fmt.Errorf("%w: device id is required", ErrValidation)
	}

	trimmedSensor := strings.TrimSpace(sensor)
	if trimmedSensor == "" {
		return fmt.Errorf("%w: sensor is required", ErrValidation)
	}

	if trigger.Min == nil && trigger.Max == nil {
		return fmt.Errorf("%w: at least one threshold is required", ErrValidation)
	}

	if trigger.Min != nil && trigger.Max != nil && *trigger.Min >= *trigger.Max {
		return fmt.Errorf("%w: min threshold must be lower than max threshold", ErrValidation)
	}

	if service.triggerRepository != nil {
		if err := service.triggerRepository.Upsert(context.Background(), trimmedDeviceID, trimmedSensor, trigger); err != nil {
			return err
		}
	}

	service.triggerMutex.Lock()
	defer service.triggerMutex.Unlock()

	if service.triggers[trimmedDeviceID] == nil {
		service.triggers[trimmedDeviceID] = make(map[string]models.SensorTrigger)
	}
	service.triggers[trimmedDeviceID][trimmedSensor] = trigger

	if service.triggerState[trimmedDeviceID] == nil {
		service.triggerState[trimmedDeviceID] = make(map[string]sensorTriggerState)
	}
	service.triggerState[trimmedDeviceID][trimmedSensor] = sensorTriggerState{}

	if service.logger != nil {
		service.logger.Info("sensor trigger configured", "deviceId", trimmedDeviceID, "sensor", trimmedSensor, "min", trigger.Min, "max", trigger.Max)
	}

	return nil
}

func (service *TelemetryService) GetSensorTrigger(_ context.Context, deviceID, sensor string) (models.SensorTrigger, error) {
	trimmedDeviceID := strings.TrimSpace(deviceID)
	if trimmedDeviceID == "" {
		return models.SensorTrigger{}, fmt.Errorf("%w: device id is required", ErrValidation)
	}

	trimmedSensor := strings.TrimSpace(sensor)
	if trimmedSensor == "" {
		return models.SensorTrigger{}, fmt.Errorf("%w: sensor is required", ErrValidation)
	}

	service.triggerMutex.RLock()
	deviceTriggers, exists := service.triggers[trimmedDeviceID]
	if exists {
		trigger, triggerExists := deviceTriggers[trimmedSensor]
		service.triggerMutex.RUnlock()
		if triggerExists {
			return trigger, nil
		}

		return models.SensorTrigger{}, repositories.ErrNotFound
	}
	service.triggerMutex.RUnlock()

	if err := service.loadDeviceTriggers(trimmedDeviceID); err != nil {
		return models.SensorTrigger{}, err
	}

	service.triggerMutex.RLock()
	deviceTriggers = service.triggers[trimmedDeviceID]
	if deviceTriggers == nil {
		service.triggerMutex.RUnlock()
		return models.SensorTrigger{}, repositories.ErrNotFound
	}

	trigger, triggerExists := deviceTriggers[trimmedSensor]
	service.triggerMutex.RUnlock()
	if !triggerExists {
		return models.SensorTrigger{}, repositories.ErrNotFound
	}

	return trigger, nil
}

func (service *TelemetryService) ListSensorTriggers(_ context.Context, deviceID string) (map[string]models.SensorTrigger, error) {
	trimmedDeviceID := strings.TrimSpace(deviceID)
	if trimmedDeviceID == "" {
		return nil, fmt.Errorf("%w: device id is required", ErrValidation)
	}

	service.triggerMutex.RLock()
	deviceTriggers, exists := service.triggers[trimmedDeviceID]
	if exists {
		defer service.triggerMutex.RUnlock()
		clonedTriggers := make(map[string]models.SensorTrigger, len(deviceTriggers))
		for sensor, trigger := range deviceTriggers {
			clonedTriggers[sensor] = trigger
		}

		return clonedTriggers, nil
	}
	service.triggerMutex.RUnlock()

	if err := service.loadDeviceTriggers(trimmedDeviceID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return map[string]models.SensorTrigger{}, nil
		}

		return nil, err
	}

	service.triggerMutex.RLock()
	defer service.triggerMutex.RUnlock()
	deviceTriggers = service.triggers[trimmedDeviceID]
	if deviceTriggers == nil {
		return map[string]models.SensorTrigger{}, nil
	}

	clonedTriggers := make(map[string]models.SensorTrigger, len(deviceTriggers))
	for sensor, trigger := range deviceTriggers {
		clonedTriggers[sensor] = trigger
	}

	return clonedTriggers, nil
}

func (service *TelemetryService) DeleteSensorTrigger(_ context.Context, deviceID, sensor string) error {
	trimmedDeviceID := strings.TrimSpace(deviceID)
	if trimmedDeviceID == "" {
		return fmt.Errorf("%w: device id is required", ErrValidation)
	}

	trimmedSensor := strings.TrimSpace(sensor)
	if trimmedSensor == "" {
		return fmt.Errorf("%w: sensor is required", ErrValidation)
	}

	if service.triggerRepository != nil {
		if err := service.triggerRepository.DeleteByDeviceAndSensor(context.Background(), trimmedDeviceID, trimmedSensor); err != nil {
			return err
		}
	}

	service.triggerMutex.Lock()
	defer service.triggerMutex.Unlock()

	if service.triggers[trimmedDeviceID] != nil {
		delete(service.triggers[trimmedDeviceID], trimmedSensor)
	}

	if service.triggerState[trimmedDeviceID] != nil {
		delete(service.triggerState[trimmedDeviceID], trimmedSensor)
	}

	if service.logger != nil {
		service.logger.Info("sensor trigger deleted", "deviceId", trimmedDeviceID, "sensor", trimmedSensor)
	}

	return nil
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

	service.evaluateSensorTriggers(context, reading)

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

func (service *TelemetryService) evaluateSensorTriggers(context context.Context, reading models.TelemetryReading) {
	service.triggerMutex.RLock()
	deviceTriggers, exists := service.triggers[reading.DeviceID]
	if !exists && service.triggerRepository != nil {
		service.triggerMutex.RUnlock()
		if err := service.loadDeviceTriggers(reading.DeviceID); err != nil {
			if service.logger != nil && !errors.Is(err, repositories.ErrNotFound) {
				service.logger.Error("load device triggers failed", "deviceId", reading.DeviceID, "error", err)
			}
			return
		}
		service.triggerMutex.RLock()
		deviceTriggers = service.triggers[reading.DeviceID]
		exists = deviceTriggers != nil
	}

	if !exists {
		service.triggerMutex.RUnlock()
		return
	}

	stateBySensor := service.triggerState[reading.DeviceID]
	triggersCopy := make(map[string]models.SensorTrigger, len(deviceTriggers))
	statesCopy := make(map[string]sensorTriggerState, len(deviceTriggers))
	for sensor, trigger := range deviceTriggers {
		triggersCopy[sensor] = trigger
		statesCopy[sensor] = stateBySensor[sensor]
	}
	service.triggerMutex.RUnlock()

	for sensor, trigger := range triggersCopy {
		value, hasSensor := reading.Sensors[sensor]
		if !hasSensor {
			continue
		}

		state := statesCopy[sensor]

		if trigger.Min != nil {
			if value <= *trigger.Min {
				if !state.MinActive {
					service.sendActivation(context, reading.DeviceID, sensor, "below_min", "min", value, *trigger.Min)
					state.MinActive = true
				}
			} else {
				state.MinActive = false
			}
		}

		if trigger.Max != nil {
			if value >= *trigger.Max {
				if !state.MaxActive {
					service.sendActivation(context, reading.DeviceID, sensor, "above_max", "max", value, *trigger.Max)
					state.MaxActive = true
				}
			} else {
				state.MaxActive = false
			}
		}

		statesCopy[sensor] = state
	}

	service.triggerMutex.Lock()
	if service.triggerState[reading.DeviceID] == nil {
		service.triggerState[reading.DeviceID] = make(map[string]sensorTriggerState)
	}
	for sensor, state := range statesCopy {
		service.triggerState[reading.DeviceID][sensor] = state
	}
	service.triggerMutex.Unlock()
}

func (service *TelemetryService) sendActivation(context context.Context, deviceID, sensor, triggerType, limitType string, value, threshold float64) {
	if service.triggerPublisher == nil {
		if service.logger != nil {
			service.logger.Warn("activation not sent: trigger publisher not configured", "deviceId", deviceID, "triggerType", triggerType)
		}
		return
	}

	command := mqtt.ActivationCommand{
		DeviceID:  deviceID,
		Trigger:   triggerType,
		Sensor:    sensor,
		LimitType: limitType,
		Value:     value,
		Threshold: threshold,
		Activated: true,
		Timestamp: time.Now().UTC().Unix(),
	}

	if err := service.triggerPublisher.PublishActivationCommand(context, command); err != nil {
		if service.logger != nil {
			service.logger.Error("activation publish failed", "deviceId", deviceID, "triggerType", triggerType, "error", err)
		}
		return
	}

	if service.logger != nil {
		service.logger.Info("activation command published", "deviceId", deviceID, "triggerType", triggerType, "value", value, "threshold", threshold)
	}
}

func (service *TelemetryService) loadDeviceTriggers(deviceID string) error {
	if service.triggerRepository == nil {
		return repositories.ErrNotFound
	}

	triggers, err := service.triggerRepository.ListByDeviceID(context.Background(), deviceID)
	if err != nil {
		return err
	}

	service.triggerMutex.Lock()
	defer service.triggerMutex.Unlock()

	if len(triggers) == 0 {
		service.triggers[deviceID] = map[string]models.SensorTrigger{}
		if service.triggerState[deviceID] == nil {
			service.triggerState[deviceID] = map[string]sensorTriggerState{}
		}
		return nil
	}

	service.triggers[deviceID] = triggers
	if service.triggerState[deviceID] == nil {
		service.triggerState[deviceID] = make(map[string]sensorTriggerState, len(triggers))
	}
	for sensor := range triggers {
		if _, exists := service.triggerState[deviceID][sensor]; !exists {
			service.triggerState[deviceID][sensor] = sensorTriggerState{}
		}
	}

	return nil
}
