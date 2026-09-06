package realtime

import (
	"log/slog"
	"sync"

	"agronode/backend/internal/models"
)

type Hub struct {
	logger             *slog.Logger
	mutex              sync.RWMutex
	telemetryClients   map[chan models.TelemetryReading]struct{}
	statusEventClients map[chan models.DeviceStatusEvent]struct{}
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		logger:             logger,
		telemetryClients:   make(map[chan models.TelemetryReading]struct{}),
		statusEventClients: make(map[chan models.DeviceStatusEvent]struct{}),
	}
}

func (hub *Hub) Subscribe() chan models.TelemetryReading {
	channel := make(chan models.TelemetryReading, 32)

	hub.mutex.Lock()
	hub.telemetryClients[channel] = struct{}{}
	hub.mutex.Unlock()

	if hub.logger != nil {
		hub.logger.Debug("websocket telemetry subscriber added")
	}

	return channel
}

func (hub *Hub) Unsubscribe(channel chan models.TelemetryReading) {
	hub.mutex.Lock()
	if _, exists := hub.telemetryClients[channel]; exists {
		delete(hub.telemetryClients, channel)
		close(channel)
	}
	hub.mutex.Unlock()

	if hub.logger != nil {
		hub.logger.Debug("websocket telemetry subscriber removed")
	}
}

func (hub *Hub) Publish(reading models.TelemetryReading) {
	hub.mutex.RLock()
	defer hub.mutex.RUnlock()

	for channel := range hub.telemetryClients {
		select {
		case channel <- reading:
		default:
		}
	}
}

func (hub *Hub) SubscribeDeviceEvents() chan models.DeviceStatusEvent {
	channel := make(chan models.DeviceStatusEvent, 32)

	hub.mutex.Lock()
	hub.statusEventClients[channel] = struct{}{}
	hub.mutex.Unlock()

	if hub.logger != nil {
		hub.logger.Debug("websocket device event subscriber added")
	}

	return channel
}

func (hub *Hub) UnsubscribeDeviceEvents(channel chan models.DeviceStatusEvent) {
	hub.mutex.Lock()
	if _, exists := hub.statusEventClients[channel]; exists {
		delete(hub.statusEventClients, channel)
		close(channel)
	}
	hub.mutex.Unlock()

	if hub.logger != nil {
		hub.logger.Debug("websocket device event subscriber removed")
	}
}

// PublishDeviceEvent publishes a device status event to all subscribers
func (hub *Hub) PublishDeviceEvent(event models.DeviceStatusEvent) {
	hub.mutex.RLock()
	defer hub.mutex.RUnlock()

	for channel := range hub.statusEventClients {
		select {
		case channel <- event:
		default:
		}
	}
}
