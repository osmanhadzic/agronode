package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"agronode/backend/internal/fuzzy"
	"agronode/backend/internal/models"
	"agronode/backend/internal/mqtt"
	"agronode/backend/internal/repositories"
)

type TelemetryService struct {
	repository             repositories.TelemetryRepository
	triggerRepository      repositories.TriggerRepository
	sensorRepository       repositories.SensorRepository
	logger                 *slog.Logger
	broadcaster            TelemetryBroadcaster
	triggerPublisher       TriggerCommandPublisher
	presenceUpdater        DevicePresenceUpdater
	sensorDiscoveryUpdater DeviceSensorDiscoveryUpdater
	metadataUpdater        DeviceMetadataUpdater
	triggers               map[string]map[string]models.SensorTrigger
	triggerState           map[string]map[string]sensorTriggerState
	triggerMutex           sync.RWMutex
}

var ErrValidation = errors.New("telemetry validation failed")

type TelemetryBroadcaster interface {
	Publish(models.TelemetryReading)
}

type TriggerCommandPublisher interface {
	PublishActivationCommand(context.Context, mqtt.ActivationCommand) error
}

type DeviceMetadataUpdater interface {
	UpdateMetadataFromTelemetry(ctx context.Context, deviceID string, meta *models.DeviceMeta) error
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

func (service *TelemetryService) SetPresenceUpdater(updater DevicePresenceUpdater) {
	service.presenceUpdater = updater
}

func (service *TelemetryService) SetSensorDiscoveryUpdater(updater DeviceSensorDiscoveryUpdater) {
	service.sensorDiscoveryUpdater = updater
}

func (service *TelemetryService) SetMetadataUpdater(updater DeviceMetadataUpdater) {
	service.metadataUpdater = updater
}

func (service *TelemetryService) SetTriggerPublisher(publisher TriggerCommandPublisher) {
	service.triggerPublisher = publisher
}

func (service *TelemetryService) SetTriggerRepository(repository repositories.TriggerRepository) {
	service.triggerRepository = repository
}

// SetSensorRepository sets the optional sensor repository used to persist trigger sensors.
func (service *TelemetryService) SetSensorRepository(repository repositories.SensorRepository) {
	service.sensorRepository = repository
}

func (service *TelemetryService) SetSensorTrigger(ctx context.Context, deviceID, sensorID string, trigger models.SensorTrigger) error {
	trimmedDeviceID := strings.TrimSpace(deviceID)
	if trimmedDeviceID == "" {
		return fmt.Errorf("%w: device id is required", ErrValidation)
	}

	trimmedSensorID := strings.TrimSpace(sensorID)
	if trimmedSensorID == "" {
		return fmt.Errorf("%w: sensor id is required", ErrValidation)
	}

	// allow fuzzy-only triggers when fuzzy config is provided
	if trigger.Min == nil && trigger.Max == nil && len(trigger.FuzzyConfig) == 0 {
		return fmt.Errorf("%w: at least one threshold or fuzzyConfig is required", ErrValidation)
	}

	if trigger.Min != nil && trigger.Max != nil && *trigger.Min >= *trigger.Max {
		return fmt.Errorf("%w: min threshold must be lower than max threshold", ErrValidation)
	}

	if strings.TrimSpace(trigger.TargetDeviceID) == "" {
		trigger.TargetDeviceID = trimmedDeviceID
	} else {
		trigger.TargetDeviceID = strings.TrimSpace(trigger.TargetDeviceID)
	}

	if service.sensorRepository != nil {
		if err := service.sensorRepository.Upsert(ctx, trimmedDeviceID, trimmedSensorID); err != nil {
			return err
		}
	}

	// validate fuzzy config if present
	if len(trigger.FuzzyConfig) > 0 {
		var ft fuzzy.FuzzyTrigger
		if err := json.Unmarshal(trigger.FuzzyConfig, &ft); err != nil {
			return fmt.Errorf("%w: fuzzyConfig invalid json", ErrValidation)
		}
		if err := fuzzy.ValidateFuzzyTrigger(ft); err != nil {
			return fmt.Errorf("%w: %v", ErrValidation, err)
		}
	}

	if service.triggerRepository != nil {
		if err := service.triggerRepository.Upsert(ctx, trimmedDeviceID, trimmedSensorID, trigger); err != nil {
			return err
		}
	}

	service.triggerMutex.Lock()
	defer service.triggerMutex.Unlock()

	if service.triggers[trimmedDeviceID] == nil {
		service.triggers[trimmedDeviceID] = make(map[string]models.SensorTrigger)
	}
	service.triggers[trimmedDeviceID][trimmedSensorID] = trigger

	if service.triggerState[trimmedDeviceID] == nil {
		service.triggerState[trimmedDeviceID] = make(map[string]sensorTriggerState)
	}
	service.triggerState[trimmedDeviceID][trimmedSensorID] = sensorTriggerState{}

	if service.logger != nil {
		service.logger.Info("sensor trigger configured", "deviceId", trimmedDeviceID, "sensorId", trimmedSensorID, "min", trigger.Min, "max", trigger.Max)
	}

	return nil
}

func (service *TelemetryService) GetSensorTrigger(ctx context.Context, deviceID, sensorID string) (models.SensorTrigger, error) {
	trimmedDeviceID := strings.TrimSpace(deviceID)
	if trimmedDeviceID == "" {
		return models.SensorTrigger{}, fmt.Errorf("%w: device id is required", ErrValidation)
	}

	trimmedSensorID := strings.TrimSpace(sensorID)
	if trimmedSensorID == "" {
		return models.SensorTrigger{}, fmt.Errorf("%w: sensor id is required", ErrValidation)
	}

	service.triggerMutex.RLock()
	deviceTriggers, exists := service.triggers[trimmedDeviceID]
	if exists {
			trigger, triggerExists := deviceTriggers[trimmedSensorID]
		service.triggerMutex.RUnlock()
		if triggerExists {
			return trigger, nil
		}

		return models.SensorTrigger{}, repositories.ErrNotFound
	}
	service.triggerMutex.RUnlock()

	if err := service.loadDeviceTriggers(ctx, trimmedDeviceID); err != nil {
		return models.SensorTrigger{}, err
	}

	service.triggerMutex.RLock()
	deviceTriggers = service.triggers[trimmedDeviceID]
	if deviceTriggers == nil {
		service.triggerMutex.RUnlock()
		return models.SensorTrigger{}, repositories.ErrNotFound
	}

	trigger, triggerExists := deviceTriggers[trimmedSensorID]
	service.triggerMutex.RUnlock()
	if !triggerExists {
		return models.SensorTrigger{}, repositories.ErrNotFound
	}

	return trigger, nil
}

func (service *TelemetryService) ListSensorTriggers(ctx context.Context, deviceID string) (map[string]models.SensorTrigger, error) {
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

	if err := service.loadDeviceTriggers(ctx, trimmedDeviceID); err != nil {
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

func (service *TelemetryService) DeleteSensorTrigger(ctx context.Context, deviceID, sensorID string) error {
	trimmedDeviceID := strings.TrimSpace(deviceID)
	if trimmedDeviceID == "" {
		return fmt.Errorf("%w: device id is required", ErrValidation)
	}

	trimmedSensorID := strings.TrimSpace(sensorID)
	if trimmedSensorID == "" {
		return fmt.Errorf("%w: sensor id is required", ErrValidation)
	}

	if service.triggerRepository != nil {
		if err := service.triggerRepository.DeleteByDeviceAndSensor(ctx, trimmedDeviceID, trimmedSensorID); err != nil {
			return err
		}
	}

	service.triggerMutex.Lock()
	defer service.triggerMutex.Unlock()

	if service.triggers[trimmedDeviceID] != nil {
		delete(service.triggers[trimmedDeviceID], trimmedSensorID)
	}

	if service.triggerState[trimmedDeviceID] != nil {
		delete(service.triggerState[trimmedDeviceID], trimmedSensorID)
	}

	if service.logger != nil {
		service.logger.Info("sensor trigger deleted", "deviceId", trimmedDeviceID, "sensorId", trimmedSensorID)
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

	resolvedSensorID := resolveTelemetrySensorID(reading)
	if resolvedSensorID == "" {
		return fmt.Errorf("%w: sensor id is required", ErrValidation)
	}
	reading.SensorID = resolvedSensorID

	if service.sensorRepository != nil {
		if err := service.sensorRepository.Upsert(context, strings.TrimSpace(reading.DeviceID), resolvedSensorID); err != nil {
			return err
		}
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

func resolveTelemetrySensorID(reading models.TelemetryReading) string {
	trimmedSensorID := strings.TrimSpace(reading.SensorID)
	if trimmedSensorID != "" {
		return trimmedSensorID
	}

	if len(reading.Sensors) != 1 {
		return ""
	}

	for sensorID := range reading.Sensors {
		trimmedSensorID = strings.TrimSpace(sensorID)
		if trimmedSensorID == "" {
			return ""
		}

		return trimmedSensorID
	}

	return ""
}

func (service *TelemetryService) HandleTelemetry(context context.Context, telemetry mqtt.TelemetryEnvelope) error {
	temperature := 0.0
	humidity := 0.0
	if sensorTemperature, hasTemperature := telemetry.Sensors["temperature"]; hasTemperature {
		temperature = sensorTemperature
	}
	if sensorTemperature, hasTemperature := telemetry.Sensors["dht11-temp"]; hasTemperature {
		temperature = sensorTemperature
	}
	if sensorHumidity, hasHumidity := telemetry.Sensors["humidity"]; hasHumidity {
		humidity = sensorHumidity
	}
	if sensorHumidity, hasHumidity := telemetry.Sensors["humidity_dht11"]; hasHumidity {
		humidity = sensorHumidity
	}
	if sensorHumidity, hasHumidity := telemetry.Sensors["dht11-humidity"]; hasHumidity {
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
		SensorID:    telemetry.SensorID,
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

	metadataForDevice := reading.Meta
	if metadataForDevice == nil {
		if signalStrength, ok := reading.Sensors["signal_strength"]; ok {
			metadataForDevice = &models.DeviceMeta{RSSI: int(signalStrength)}
		}
	}

	if metadataForDevice != nil && service.metadataUpdater != nil {
		if err := service.metadataUpdater.UpdateMetadataFromTelemetry(context, reading.DeviceID, metadataForDevice); err != nil {
			if !errors.Is(err, repositories.ErrDeviceNotFound) {
				if service.logger != nil {
					service.logger.Warn("device metadata update failed", "deviceId", reading.DeviceID, "error", err)
				}
			} else if service.logger != nil {
				service.logger.Warn("device metadata update skipped: device not registered", "deviceId", reading.DeviceID)
			}
		}
	}

	if service.presenceUpdater != nil {
		if err := service.presenceUpdater.UpdatePresence(context, reading.DeviceID, reading.CreatedAt); err != nil {
			if !errors.Is(err, repositories.ErrDeviceNotFound) {
				if service.logger != nil {
					service.logger.Warn("device presence update failed", "deviceId", reading.DeviceID, "error", err)
				}
			} else if service.logger != nil {
				service.logger.Warn("device presence skipped: device not registered", "deviceId", reading.DeviceID)
			}
		}
	}

	if service.sensorDiscoveryUpdater != nil {
		sensorNames := make([]string, 0, len(telemetry.Sensors))
		for key := range telemetry.Sensors {
			sensorNames = append(sensorNames, key)
		}

		if err := service.sensorDiscoveryUpdater.UpdateDiscoveredSensors(context, reading.DeviceID, sensorNames); err != nil {
			if service.logger != nil {
				service.logger.Warn("device sensor discovery update failed", "deviceId", reading.DeviceID, "error", err)
			}
		}
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

func (service *TelemetryService) GetTelemetryByDeviceIDAndSensorID(context context.Context, deviceID, sensorID string) ([]models.TelemetryReading, error) {
	if service.repository == nil {
		return nil, errors.New("telemetry repository is not configured")
	}

	if strings.TrimSpace(deviceID) == "" {
		return nil, fmt.Errorf("%w: device id is required", ErrValidation)
	}

	if strings.TrimSpace(sensorID) == "" {
		return nil, fmt.Errorf("%w: sensor id is required", ErrValidation)
	}

	return service.repository.ListByDeviceIDAndSensorID(context, deviceID, sensorID)
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

func (service *TelemetryService) evaluateSensorTriggers(context context.Context, reading models.TelemetryReading) {
	service.triggerMutex.RLock()
	deviceTriggers, exists := service.triggers[reading.DeviceID]
	if !exists && service.triggerRepository != nil {
		service.triggerMutex.RUnlock()
		if err := service.loadDeviceTriggers(context, reading.DeviceID); err != nil {
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
	for sensorID, trigger := range deviceTriggers {
		triggersCopy[sensorID] = trigger
		statesCopy[sensorID] = stateBySensor[sensorID]
	}
	service.triggerMutex.RUnlock()

	for sensorID, trigger := range triggersCopy {
		value, hasSensor := reading.Sensors[sensorID]
		if !hasSensor {
			continue
		}

		state := statesCopy[sensorID]

		if trigger.Min != nil {
			if value <= *trigger.Min {
				if !state.MinActive {
					targetDeviceID := resolveTargetDeviceID(reading.DeviceID, trigger)
					service.sendActivation(context, reading.DeviceID, targetDeviceID, sensorID, "below_min", "min", value, *trigger.Min)
					state.MinActive = true
				}
			} else {
				state.MinActive = false
			}
		}

		if trigger.Max != nil {
			if value >= *trigger.Max {
				if !state.MaxActive {
					targetDeviceID := resolveTargetDeviceID(reading.DeviceID, trigger)
					service.sendActivation(context, reading.DeviceID, targetDeviceID, sensorID, "above_max", "max", value, *trigger.Max)
					state.MaxActive = true
				}
			} else {
				state.MaxActive = false
			}
		}

		// Fuzzy trigger evaluation (if fuzzy config present)
		if len(trigger.FuzzyConfig) > 0 {
			var fTrigger fuzzy.FuzzyTrigger
			if err := json.Unmarshal(trigger.FuzzyConfig, &fTrigger); err != nil {
				if service.logger != nil {
					service.logger.Warn("invalid fuzzy trigger config, skipping", "deviceId", reading.DeviceID, "sensorId", sensorID, "error", err)
				}
			} else {
				// Prepare inputs: use full reading.Sensors map
				inputs := make(map[string]float64, len(reading.Sensors))
				for k, v := range reading.Sensors {
					inputs[k] = v
				}

				// If membership functions don't specify sensor, assume current sensor
				for i, mf := range fTrigger.MembershipFunctions {
					if mf.Sensor == "" {
						fTrigger.MembershipFunctions[i].Sensor = sensorID
					}
				}

				eval, err := fuzzy.EvaluateTrigger(fTrigger, inputs)
				if err != nil {
					if service.logger != nil {
						service.logger.Warn("fuzzy evaluation failed", "deviceId", reading.DeviceID, "sensorId", sensorID, "error", err)
					}
				} else {
					// Log membership degrees (sparse)
					if service.logger != nil {
						service.logger.Debug("fuzzy evaluation", "deviceId", reading.DeviceID, "sensorId", sensorID, "memberships", eval.Memberships, "rules", eval.Rules)
					}

					// For each rule result with strength > 0, send activation command
					for _, rr := range eval.Rules {
						if rr.Strength <= 0 {
							continue
						}

						targetDeviceID := resolveTargetDeviceID(reading.DeviceID, trigger)

						// Use trigger name including rule for clarity
						cmd := mqtt.ActivationCommand{
							DeviceID:  targetDeviceID,
							Trigger:   "fuzzy:" + rr.Name,
							Sensor:    sensorID,
							LimitType: rr.Action.Type,
							Value:     rr.Value,
							Threshold: rr.Strength,
							Activated: true,
							Timestamp: time.Now().UTC().Unix(),
						}

						reason := fmt.Sprintf("source_device=%s sensor=%s action=%s value=%.4f strength=%.4f", reading.DeviceID, sensorID, rr.Action.Type, rr.Value, rr.Strength)
						service.publishActivationCommand(context, cmd, reason)
					}
				}
			}
		}

		statesCopy[sensorID] = state
	}
	service.triggerMutex.Lock()
	if service.triggerState[reading.DeviceID] == nil {
		service.triggerState[reading.DeviceID] = make(map[string]sensorTriggerState)
	}
	for sensorID, state := range statesCopy {
		service.triggerState[reading.DeviceID][sensorID] = state
	}
	service.triggerMutex.Unlock()
}

func (service *TelemetryService) sendActivation(context context.Context, sourceDeviceID, deviceID, sensorID, triggerType, limitType string, value, threshold float64) {

	command := mqtt.ActivationCommand{
		DeviceID:  deviceID,
		Trigger:   triggerType,
		Sensor:    sensorID,
		LimitType: limitType,
		Value:     value,
		Threshold: threshold,
		Activated: true,
		Timestamp: time.Now().UTC().Unix(),
	}

	reason := fmt.Sprintf("source_device=%s sensor=%s limit=%s value=%.4f threshold=%.4f", sourceDeviceID, sensorID, limitType, value, threshold)
	service.publishActivationCommand(context, command, reason)
}

func (service *TelemetryService) publishActivationCommand(context context.Context, command mqtt.ActivationCommand, reason string) {
	if service.triggerPublisher == nil {
		if service.logger != nil {
			service.logger.Warn("activation not sent: trigger publisher not configured", "deviceId", command.DeviceID, "triggerType", command.Trigger)
		}
		service.persistExecutionAttempt(context, command, reason, models.IrrigationExecutionStatusFailed, errors.New("trigger publisher not configured"))
		return
	}

	if err := service.triggerPublisher.PublishActivationCommand(context, command); err != nil {
		if service.logger != nil {
			service.logger.Error("activation publish failed", "deviceId", command.DeviceID, "triggerType", command.Trigger, "error", err)
		}
		service.persistExecutionAttempt(context, command, reason, models.IrrigationExecutionStatusFailed, err)
		return
	}

	service.persistExecutionAttempt(context, command, reason, models.IrrigationExecutionStatusSucceeded, nil)

	if service.logger != nil {
		service.logger.Info("activation command published", "deviceId", command.DeviceID, "triggerType", command.Trigger, "value", command.Value, "threshold", command.Threshold)
	}
}

func (service *TelemetryService) persistExecutionAttempt(context context.Context, command mqtt.ActivationCommand, reason, status string, attemptError error) {
	if service.triggerRepository == nil {
		return
	}

	triggeredAt := time.Now().UTC()
	if command.Timestamp > 0 {
		triggeredAt = time.Unix(command.Timestamp, 0).UTC()
	}

	execution := models.IrrigationExecution{
		RuleID:      command.Trigger,
		DeviceID:    command.DeviceID,
		TriggeredAt: triggeredAt,
		Reason:      reason,
		Status:      status,
	}

	if attemptError != nil {
		errorMessage := attemptError.Error()
		execution.ErrorMessage = &errorMessage
	}

	if err := service.triggerRepository.SaveExecution(context, execution); err != nil {
		if service.logger != nil {
			service.logger.Warn("persist trigger execution failed", "deviceId", command.DeviceID, "ruleId", command.Trigger, "error", err)
		}
		return
	}

	if service.logger != nil {
		if execution.ErrorMessage != nil {
			service.logger.Info("trigger execution persisted", "deviceId", execution.DeviceID, "ruleId", execution.RuleID, "status", execution.Status, "reason", execution.Reason, "error", *execution.ErrorMessage)
			return
		}

		service.logger.Info("trigger execution persisted", "deviceId", execution.DeviceID, "ruleId", execution.RuleID, "status", execution.Status, "reason", execution.Reason)
	}
}

func (service *TelemetryService) loadDeviceTriggers(context context.Context, deviceID string) error {
	if service.triggerRepository == nil {
		return repositories.ErrNotFound
	}

	triggers, err := service.triggerRepository.ListByDeviceID(context, deviceID)
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
	for sensorID, trigger := range service.triggers[deviceID] {
		service.triggers[deviceID][sensorID] = normalizeTriggerTarget(deviceID, trigger)
	}
	if service.triggerState[deviceID] == nil {
		service.triggerState[deviceID] = make(map[string]sensorTriggerState, len(triggers))
	}
	for sensorID := range triggers {
		if _, exists := service.triggerState[deviceID][sensorID]; !exists {
			service.triggerState[deviceID][sensorID] = sensorTriggerState{}
		}
	}

	return nil
}

func resolveTargetDeviceID(sourceDeviceID string, trigger models.SensorTrigger) string {
	trimmedTarget := strings.TrimSpace(trigger.TargetDeviceID)
	if trimmedTarget != "" {
		return trimmedTarget
	}

	return sourceDeviceID
}

func normalizeTriggerTarget(deviceID string, trigger models.SensorTrigger) models.SensorTrigger {
	if strings.TrimSpace(trigger.TargetDeviceID) == "" {
		trigger.TargetDeviceID = deviceID
		return trigger
	}

	trigger.TargetDeviceID = strings.TrimSpace(trigger.TargetDeviceID)
	return trigger
}
