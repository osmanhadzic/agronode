package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"agronode/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type TelemetryRealtimeHub interface {
	Subscribe() chan models.TelemetryReading
	Unsubscribe(chan models.TelemetryReading)
}

type realtimeHandler struct {
	logger   *slog.Logger
	hub      TelemetryRealtimeHub
	upgrader websocket.Upgrader
}

func RegisterRealtimeRoutes(router *gin.Engine, logger *slog.Logger, hub TelemetryRealtimeHub) {
	handler := &realtimeHandler{
		logger: logger,
		hub:    hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(request *http.Request) bool {
				return true
			},
		},
	}

	router.GET("/ws/telemetry", handler.streamTelemetry)
}

func (handler *realtimeHandler) streamTelemetry(context *gin.Context) {
	if handler.hub == nil {
		context.JSON(http.StatusServiceUnavailable, gin.H{"error": "realtime hub unavailable"})
		return
	}

	connection, err := handler.upgrader.Upgrade(context.Writer, context.Request, nil)
	if err != nil {
		if handler.logger != nil {
			handler.logger.Warn("websocket upgrade failed", "error", err)
		}
		return
	}

	connection.SetReadLimit(1024)
	connection.SetReadDeadline(time.Now().Add(60 * time.Second))
	connection.SetPongHandler(func(string) error {
		connection.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	channel := handler.hub.Subscribe()
	defer handler.hub.Unsubscribe(channel)
	defer connection.Close()

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case reading, ok := <-channel:
			if !ok {
				return
			}

			if writeErr := connection.WriteJSON(reading); writeErr != nil {
				return
			}
		case <-pingTicker.C:
			case <-pingTicker.C:
			if err := connection.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
				return
			}
		}
	}
}
