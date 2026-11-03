package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"agronode/backend/internal/repositories"
	"agronode/backend/internal/services"
	"agronode/backend/internal/tenancy"
	"github.com/gin-gonic/gin"
)

type DeviceProvisioningService interface {
	IssueToken(ctx context.Context, organizationID uint, assetUnitID *uint, ttl time.Duration, createdBy string) (*services.IssuedProvisioningToken, error)
	Register(ctx context.Context, hardwareID string, token string) (*services.DeviceRegistration, error)
	RotateCredentials(ctx context.Context, deviceID string) (*services.DeviceRegistration, error)
	SetLifecycleStatus(ctx context.Context, deviceID string, status string) error
}

type ProvisioningOptions struct {
	RequireHTTPS bool
	// RegisterRateLimit is the maximum registration attempts per client IP per minute (0 disables).
	RegisterRateLimit int
}

type provisioningHandler struct {
	logger  *slog.Logger
	service DeviceProvisioningService
	options ProvisioningOptions
	limiter *attemptLimiter
}

type deviceRegistrationRequest struct {
	HardwareID        string `json:"hardwareId" binding:"required"`
	ProvisioningToken string `json:"provisioningToken" binding:"required"`
}

type mqttCredentialsResponse struct {
	Username string `json:"username"`
	ClientID string `json:"clientId"`
	Password string `json:"password"`
}

type deviceRegistrationResponse struct {
	DeviceID       string                  `json:"deviceId"`
	HardwareID     string                  `json:"hardwareId,omitempty"`
	OrganizationID *uint                   `json:"organizationId,omitempty"`
	AssetUnitID    *uint                   `json:"assetUnitId,omitempty"`
	MQTT           mqttCredentialsResponse `json:"mqtt"`
}

type issueProvisioningTokenRequest struct {
	OrganizationID uint  `json:"organizationId"`
	AssetUnitID    *uint `json:"assetUnitId"`
	TTLHours       int   `json:"ttlHours"`
}

type lifecycleRequest struct {
	Status string `json:"status" binding:"required"`
}

// RegisterPublicProvisioningRoutes mounts the unauthenticated, token-gated registration endpoint.
func RegisterPublicProvisioningRoutes(router gin.IRoutes, logger *slog.Logger, service DeviceProvisioningService, options ProvisioningOptions) {
	handler := &provisioningHandler{logger: logger, service: service, options: options, limiter: newAttemptLimiter(options.RegisterRateLimit, time.Minute)}
	router.POST("/api/v1/devices/register", handler.register)
}

// RegisterProvisioningAdminRoutes mounts session-protected token issuance and recovery routes.
func RegisterProvisioningAdminRoutes(api *gin.RouterGroup, logger *slog.Logger, service DeviceProvisioningService) {
	handler := &provisioningHandler{logger: logger, service: service}
	api.POST("/devices/provisioning-tokens", handler.issueToken)
	api.POST("/devices/:deviceId/credentials/rotate", handler.rotateCredentials)
	api.PATCH("/devices/:deviceId/lifecycle", handler.setLifecycle)
}

func (handler *provisioningHandler) register(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")

	if handler.options.RequireHTTPS && !requestIsHTTPS(ctx.Request) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "https is required"})
		return
	}

	if !handler.limiter.allow(ctx.ClientIP()) {
		ctx.JSON(http.StatusTooManyRequests, gin.H{"error": "too many registration attempts"})
		return
	}

	var req deviceRegistrationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	result, err := handler.service.Register(ctx.Request.Context(), req.HardwareID, req.ProvisioningToken)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrDeviceValidation):
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, repositories.ErrProvisioningTokenInvalid):
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "provisioning token is invalid or expired"})
		case errors.Is(err, repositories.ErrDuplicateHardwareID):
			ctx.JSON(http.StatusConflict, gin.H{"error": "hardware id already registered; use the authorized credential recovery workflow"})
		default:
			handler.logger.Error("device provisioning failed", "error", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register device"})
		}
		return
	}

	response := deviceRegistrationResponse{
		DeviceID:       result.Device.DeviceID,
		OrganizationID: result.Device.OrganizationID,
		AssetUnitID:    result.Device.ZoneID,
		MQTT:           mqttCredentialsResponse{Username: result.MQTTUsername, ClientID: result.MQTTUsername, Password: result.MQTTPassword},
	}
	if result.Device.HardwareID != nil {
		response.HardwareID = *result.Device.HardwareID
	}
	ctx.JSON(http.StatusCreated, response)
}

func (handler *provisioningHandler) issueToken(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")

	requestContext, err := requestContextWithOrganizationScope(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req issueProvisioningTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	organizationID := req.OrganizationID
	if organizationID == 0 {
		scoped, ok := tenancy.OrganizationIDFromContext(requestContext)
		if !ok {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "organizationId is required"})
			return
		}
		organizationID = scoped
	}

	issued, err := handler.service.IssueToken(requestContext, organizationID, req.AssetUnitID, time.Duration(req.TTLHours)*time.Hour, ctx.GetString("sessionEmail"))
	if err != nil {
		handler.writeAdminError(ctx, err, "failed to issue provisioning token")
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"provisioningToken": issued.Token,
		"expiresAt":         issued.ExpiresAt.Format(time.RFC3339),
		"organizationId":    organizationID,
		"assetUnitId":       req.AssetUnitID,
	})
}

func (handler *provisioningHandler) rotateCredentials(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")

	requestContext, err := requestContextWithOrganizationScope(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := handler.service.RotateCredentials(requestContext, ctx.Param("deviceId"))
	if err != nil {
		handler.writeAdminError(ctx, err, "failed to rotate credentials")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"deviceId": ctx.Param("deviceId"),
		"mqtt":     mqttCredentialsResponse{Username: result.MQTTUsername, ClientID: result.MQTTUsername, Password: result.MQTTPassword},
	})
}

func (handler *provisioningHandler) setLifecycle(ctx *gin.Context) {
	requestContext, err := requestContextWithOrganizationScope(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req lifecycleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	if err := handler.service.SetLifecycleStatus(requestContext, ctx.Param("deviceId"), strings.ToLower(strings.TrimSpace(req.Status))); err != nil {
		handler.writeAdminError(ctx, err, "failed to update device status")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"deviceId": ctx.Param("deviceId"), "lifecycleStatus": strings.ToLower(strings.TrimSpace(req.Status))})
}

func (handler *provisioningHandler) writeAdminError(ctx *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, services.ErrDeviceValidation):
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrProvisioningForbidden):
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, repositories.ErrDeviceNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{"error": "device not found or not eligible"})
	default:
		handler.logger.Error(fallback, "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fallback})
	}
}

func requestIsHTTPS(request *http.Request) bool {
	return request.TLS != nil || strings.EqualFold(request.Header.Get("X-Forwarded-Proto"), "https")
}

// attemptLimiter is a small fixed-window per-key limiter; limit <= 0 disables it.
type attemptLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	counts map[string]int
	reset  time.Time
}

func newAttemptLimiter(limit int, window time.Duration) *attemptLimiter {
	return &attemptLimiter{limit: limit, window: window, counts: map[string]int{}}
}

func (limiter *attemptLimiter) allow(key string) bool {
	if limiter == nil || limiter.limit <= 0 {
		return true
	}
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := time.Now()
	if now.After(limiter.reset) {
		limiter.counts = map[string]int{}
		limiter.reset = now.Add(limiter.window)
	}
	limiter.counts[key]++
	return limiter.counts[key] <= limiter.limit
}
