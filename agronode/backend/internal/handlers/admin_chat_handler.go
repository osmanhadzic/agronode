package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"agronode/backend/internal/models"
	"agronode/backend/internal/services"
	"agronode/backend/internal/tenancy"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type AdminChatMessage struct {
	ID     string `json:"id"`
	Sender string `json:"sender"`
	Text   string `json:"text"`
	SentAt string `json:"sentAt"`
}

type adminChatInbound struct {
	Text string `json:"text"`
}

type adminChatHub struct {
	mutex   sync.RWMutex
	clients map[chan AdminChatMessage]struct{}
}

func newAdminChatHub() *adminChatHub {
	return &adminChatHub{
		clients: make(map[chan AdminChatMessage]struct{}),
	}
}

func (hub *adminChatHub) subscribe() chan AdminChatMessage {
	channel := make(chan AdminChatMessage, 32)

	hub.mutex.Lock()
	hub.clients[channel] = struct{}{}
	hub.mutex.Unlock()

	return channel
}

func (hub *adminChatHub) unsubscribe(channel chan AdminChatMessage) {
	hub.mutex.Lock()
	if _, exists := hub.clients[channel]; exists {
		delete(hub.clients, channel)
		close(channel)
	}
	hub.mutex.Unlock()
}

func (hub *adminChatHub) broadcast(message AdminChatMessage) {
	hub.mutex.RLock()
	defer hub.mutex.RUnlock()

	for channel := range hub.clients {
		select {
		case channel <- message:
		default:
		}
	}
}

type adminChatHandler struct {
	logger   *slog.Logger
	hub      *adminChatHub
	llm      services.AdminChatLLMService
	upgrader websocket.Upgrader
}

func RegisterAdminChatRoutes(router *gin.Engine, logger *slog.Logger, llm services.AdminChatLLMService) {
	if llm == nil {
		llm = services.NewNoopAdminChatLLMService()
	}

	handler := &adminChatHandler{
		logger: logger,
		hub:    newAdminChatHub(),
		llm:    llm,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(request *http.Request) bool {
				return true
			},
		},
	}

	router.GET("/ws/admin/chat", handler.streamChat)
}

func (handler *adminChatHandler) streamChat(context *gin.Context) {
	role, ok := tenancy.UserRoleFromContext(context.Request.Context())
	if !ok || role != models.UserRoleAdmin {
		context.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	connection, err := handler.upgrader.Upgrade(context.Writer, context.Request, nil)
	if err != nil {
		if handler.logger != nil {
			handler.logger.Warn("admin chat websocket upgrade failed", "error", err)
		}
		return
	}

	connection.SetReadLimit(4 * 1024)
	connection.SetReadDeadline(time.Now().Add(60 * time.Second))
	connection.SetPongHandler(func(string) error {
		connection.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	channel := handler.hub.subscribe()
	defer handler.hub.unsubscribe(channel)
	defer connection.Close()

	done := make(chan struct{})

	go func() {
		pingTicker := time.NewTicker(30 * time.Second)
		defer pingTicker.Stop()
		defer close(done)

		for {
			select {
			case message, ok := <-channel:
				if !ok {
					return
				}

				if writeErr := connection.WriteJSON(message); writeErr != nil {
					return
				}
			case <-pingTicker.C:
				if pingErr := connection.WriteMessage(websocket.PingMessage, []byte("ping")); pingErr != nil {
					return
				}
			}
		}
	}()

	sender := strings.TrimSpace(context.GetString("sessionEmail"))
	if sender == "" {
		sender = "admin"
	}

	for {
		select {
		case <-done:
			return
		default:
		}

		var inbound adminChatInbound
		if readErr := connection.ReadJSON(&inbound); readErr != nil {
			return
		}

		text := strings.TrimSpace(inbound.Text)
		if text == "" {
			continue
		}

		if len(text) > 1000 {
			text = text[:1000]
		}

		handler.hub.broadcast(AdminChatMessage{
			ID:     randomMessageID(),
			Sender: sender,
			Text:   text,
			SentAt: time.Now().UTC().Format(time.RFC3339),
		})

		if handler.llm != nil && handler.llm.Enabled() {
			go handler.sendAssistantReply(text)
		}
	}
}

func (handler *adminChatHandler) sendAssistantReply(userText string) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	reply, err := handler.llm.GenerateReply(ctx, userText)
	if err != nil {
		message := "Assistant trenutno nije dostupan."
		errorText := strings.TrimSpace(err.Error())
		if errorText != "" {
			message = "Assistant error: " + errorText
		}

		handler.hub.broadcast(AdminChatMessage{
			ID:     randomMessageID(),
			Sender: "AgroNode Assistant",
			Text:   message,
			SentAt: time.Now().UTC().Format(time.RFC3339),
		})

		if handler.logger != nil {
			handler.logger.Warn("admin chat llm reply failed", "error", err)
		}
		return
	}

	reply = strings.TrimSpace(reply)
	if reply == "" {
		return
	}

	handler.hub.broadcast(AdminChatMessage{
		ID:     randomMessageID(),
		Sender: "AgroNode Assistant",
		Text:   reply,
		SentAt: time.Now().UTC().Format(time.RFC3339),
	})
}

func randomMessageID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}

	return hex.EncodeToString(buffer)
}
