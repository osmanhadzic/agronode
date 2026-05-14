package realtime

import (
	"log/slog"
	"sync"

	"agronode/backend/internal/models"
)

type Hub struct {
	logger  *slog.Logger
	mutex   sync.RWMutex
	clients map[chan models.TelemetryReading]struct{}
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		logger:  logger,
		clients: make(map[chan models.TelemetryReading]struct{}),
	}
}

func (hub *Hub) Subscribe() chan models.TelemetryReading {
	channel := make(chan models.TelemetryReading, 32)

	hub.mutex.Lock()
	hub.clients[channel] = struct{}{}
	hub.mutex.Unlock()

	if hub.logger != nil {
		hub.logger.Debug("websocket subscriber added")
	}

	return channel
}

func (hub *Hub) Unsubscribe(channel chan models.TelemetryReading) {
	hub.mutex.Lock()
	if _, exists := hub.clients[channel]; exists {
		delete(hub.clients, channel)
		close(channel)
	}
	hub.mutex.Unlock()

	if hub.logger != nil {
		hub.logger.Debug("websocket subscriber removed")
	}
}

func (hub *Hub) Publish(reading models.TelemetryReading) {
	hub.mutex.RLock()
	defer hub.mutex.RUnlock()

	for channel := range hub.clients {
		select {
		case channel <- reading:
		default:
		}
	}
}
