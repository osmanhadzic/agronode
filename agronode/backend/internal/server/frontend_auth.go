package server

import (
	"net/http"
	"strings"
	"time"

	"agronode/backend/internal/auth"
	"agronode/backend/internal/tenancy"
	"github.com/gin-gonic/gin"
)

func sessionAuthMiddleware(sessionSecret string) gin.HandlerFunc {
	return func(context *gin.Context) {
		claims, err := validateSessionFromRequest(context.Request, sessionSecret)
		if err != nil {
			context.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		context.Request = context.Request.WithContext(tenancy.WithOrganizationID(context.Request.Context(), claims.OrganizationID))
		context.Set("sessionEmail", claims.Email)
		context.Next()
	}
}

func validateSessionFromRequest(request *http.Request, sessionSecret string) (auth.SessionClaims, error) {
	token := extractSessionToken(request)
	if token == "" {
		return auth.SessionClaims{}, auth.ErrInvalidToken
	}

	return auth.ValidateSessionToken(sessionSecret, token, time.Now().UTC())
}

func extractSessionToken(request *http.Request) string {
	authorizationHeader := strings.TrimSpace(request.Header.Get("Authorization"))
	if authorizationHeader != "" {
		const bearerPrefix = "Bearer "
		if strings.HasPrefix(authorizationHeader, bearerPrefix) {
			return strings.TrimSpace(strings.TrimPrefix(authorizationHeader, bearerPrefix))
		}
	}

	if token := strings.TrimSpace(request.URL.Query().Get("token")); token != "" {
		return token
	}

	if token := strings.TrimSpace(request.Header.Get("X-Frontend-Token")); token != "" {
		return token
	}

	return ""
}
