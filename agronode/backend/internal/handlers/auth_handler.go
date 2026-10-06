package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"agronode/backend/internal/auth"
	"github.com/gin-gonic/gin"
)

type AuthService interface {
	Login(email, password string, now time.Time) (auth.SessionClaims, string, error)
}

type authHandler struct {
	logger  *slog.Logger
	service AuthService
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	Token          string `json:"token"`
	Email          string `json:"email"`
	OrganizationID uint   `json:"organizationId"`
	Role           string `json:"role"`
	ExpiresAt      int64  `json:"expiresAt"`
}

func RegisterAuthRoutes(api *gin.RouterGroup, logger *slog.Logger, service AuthService) {
	handler := &authHandler{logger: logger, service: service}
	api.POST("/auth/login", handler.login)
}

func (handler *authHandler) login(context *gin.Context) {
	var request loginRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		if handler.logger != nil {
			handler.logger.Warn("login request invalid", "error", err)
		}
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	claims, token, err := handler.service.Login(strings.TrimSpace(request.Email), request.Password, time.Now().UTC())
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			if handler.logger != nil {
				handler.logger.Warn("login rejected", "email", strings.TrimSpace(request.Email), "reason", "invalid credentials")
			}
			context.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		handler.logger.Error("login failed", "error", err)
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to login"})
		return
	}

	if handler.logger != nil {
		handler.logger.Info("login succeeded", "email", claims.Email, "organizationId", claims.OrganizationID, "role", claims.Role)
	}

	context.JSON(http.StatusOK, loginResponse{
		Token:          token,
		Email:          claims.Email,
		OrganizationID: claims.OrganizationID,
		Role:           claims.Role,
		ExpiresAt:      claims.ExpiresAt,
	})
}
