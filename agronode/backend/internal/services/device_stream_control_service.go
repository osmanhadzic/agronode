package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"agronode/backend/internal/mqtt"
)

type DeviceStreamCommandPublisher interface {
	PublishActivationCommand(context.Context, mqtt.ActivationCommand) error
}

type DeviceStreamControlService struct {
	logger    *slog.Logger
	publisher DeviceStreamCommandPublisher
}

func NewDeviceStreamControlService(logger *slog.Logger, publisher DeviceStreamCommandPublisher) *DeviceStreamControlService {
	return &DeviceStreamControlService{logger: logger, publisher: publisher}
}

func (service *DeviceStreamControlService) SetTelemetryStreaming(ctx context.Context, deviceID string, sensorID string, enabled bool) error {
	trimmedDeviceID := strings.TrimSpace(deviceID)
	if trimmedDeviceID == "" {
		return fmt.Errorf("%w: device id is required", ErrValidation)
	}

	trimmedSensorID := strings.TrimSpace(sensorID)
	if trimmedSensorID == "" {
		trimmedSensorID = "all"
	}

	if service.publisher == nil {
		return fmt.Errorf("stream control publisher is not configured")
	}

	triggerType := "stream_pause"
	if enabled {
		triggerType = "stream_resume"
	}

	command := mqtt.ActivationCommand{
		DeviceID:  trimmedDeviceID,
		Trigger:   triggerType,
		Sensor:    trimmedSensorID,
		LimitType: "manual",
		Value:     0,
		Threshold: 0,
		Activated: enabled,
		Timestamp: time.Now().UTC().Unix(),
	}

	if err := service.publisher.PublishActivationCommand(ctx, command); err != nil {
		return fmt.Errorf("publish stream control command: %w", err)
	}

	if service.logger != nil {
		service.logger.Info("device telemetry stream command published", "deviceId", trimmedDeviceID, "sensorId", trimmedSensorID, "enabled", enabled, "trigger", triggerType)
	}

	return nil
}
