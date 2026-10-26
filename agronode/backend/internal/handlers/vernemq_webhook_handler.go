package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"agronode/backend/internal/services"
	"github.com/gin-gonic/gin"
)

// MQTTAuthorizer is the decision logic behind the VerneMQ vmq_webhooks auth hooks.
type MQTTAuthorizer interface {
	Authenticate(ctx context.Context, username string, clientID string, password string) error
	AuthorizePublish(ctx context.Context, username string, topic string) error
	AuthorizeSubscribe(ctx context.Context, username string, topics []string) error
}

// vmq_webhooks sends the hook name in the "vernemq-hook" header and expects HTTP 200 with
// {"result":"ok"} or {"result":{"error":"..."}}. Anything else makes the broker reject the action.
type vmqRegisterPayload struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	ClientID   string `json:"client_id"`
	Mountpoint string `json:"mountpoint"`
}

type vmqPublishPayload struct {
	Username   string `json:"username"`
	ClientID   string `json:"client_id"`
	Mountpoint string `json:"mountpoint"`
	Topic      string `json:"topic"`
}

type vmqSubscribePayload struct {
	Username   string `json:"username"`
	ClientID   string `json:"client_id"`
	Mountpoint string `json:"mountpoint"`
	Topics     []struct {
		Topic string `json:"topic"`
	} `json:"topics"`
}

type vernemqWebhookHandler struct {
	logger     *slog.Logger
	authorizer MQTTAuthorizer
}

// RegisterVerneMQWebhookRoutes mounts the broker hooks; serve them only on an internal listener.
func RegisterVerneMQWebhookRoutes(router gin.IRoutes, logger *slog.Logger, authorizer MQTTAuthorizer) {
	handler := &vernemqWebhookHandler{logger: logger, authorizer: authorizer}
	router.POST("/vernemq/auth_on_register", handler.requireHook("auth_on_register"), handler.authOnRegister)
	router.POST("/vernemq/auth_on_publish", handler.requireHook("auth_on_publish"), handler.authOnPublish)
	router.POST("/vernemq/auth_on_subscribe", handler.requireHook("auth_on_subscribe"), handler.authOnSubscribe)
}

func (handler *vernemqWebhookHandler) requireHook(name string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.GetHeader("vernemq-hook") != name {
			handler.deny(ctx, "bad_request")
			ctx.Abort()
		}
	}
}

func (handler *vernemqWebhookHandler) allow(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"result": "ok"})
}

func (handler *vernemqWebhookHandler) deny(ctx *gin.Context, reason string) {
	ctx.JSON(http.StatusOK, gin.H{"result": gin.H{"error": reason}})
}

// respond turns a decision error into a denial; infrastructure errors are logged but never allow access.
func (handler *vernemqWebhookHandler) respond(ctx *gin.Context, hook string, err error) {
	if err == nil {
		handler.allow(ctx)
		return
	}
	if !errors.Is(err, services.ErrMQTTNotAuthorized) {
		handler.logger.Error("mqtt auth backend error", "hook", hook, "error", err)
		handler.deny(ctx, "internal_error")
		return
	}
	handler.deny(ctx, "not_authorized")
}

func (handler *vernemqWebhookHandler) authOnRegister(ctx *gin.Context) {
	var payload vmqRegisterPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil || payload.Mountpoint != "" {
		handler.deny(ctx, "not_authorized")
		return
	}
	handler.respond(ctx, "auth_on_register", handler.authorizer.Authenticate(ctx.Request.Context(), payload.Username, payload.ClientID, payload.Password))
}

func (handler *vernemqWebhookHandler) authOnPublish(ctx *gin.Context) {
	var payload vmqPublishPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil || payload.Mountpoint != "" {
		handler.deny(ctx, "not_authorized")
		return
	}
	handler.respond(ctx, "auth_on_publish", handler.authorizer.AuthorizePublish(ctx.Request.Context(), payload.Username, payload.Topic))
}

func (handler *vernemqWebhookHandler) authOnSubscribe(ctx *gin.Context) {
	var payload vmqSubscribePayload
	if err := ctx.ShouldBindJSON(&payload); err != nil || payload.Mountpoint != "" {
		handler.deny(ctx, "not_authorized")
		return
	}
	topics := make([]string, 0, len(payload.Topics))
	for _, entry := range payload.Topics {
		topics = append(topics, entry.Topic)
	}
	handler.respond(ctx, "auth_on_subscribe", handler.authorizer.AuthorizeSubscribe(ctx.Request.Context(), payload.Username, topics))
}
