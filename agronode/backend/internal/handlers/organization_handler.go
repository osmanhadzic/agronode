package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"agronode/backend/internal/models"
	"github.com/gin-gonic/gin"
)

type OrganizationService interface {
	ListOrganizations(ctx context.Context) ([]models.Organization, error)
}

type organizationHandler struct {
	logger  *slog.Logger
	service OrganizationService
}

type organizationResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func RegisterOrganizationRoutes(api *gin.RouterGroup, logger *slog.Logger, service OrganizationService) {
	handler := &organizationHandler{logger: logger, service: service}
	api.GET("/organizations", handler.listOrganizations)
}

func (handler *organizationHandler) listOrganizations(ctx *gin.Context) {
	requestContext, err := requestContextWithOrganizationScope(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	organizations, err := handler.service.ListOrganizations(requestContext)
	if err != nil {
		handler.logger.Error("list organizations failed", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch organizations"})
		return
	}

	responses := make([]organizationResponse, len(organizations))
	for i, organization := range organizations {
		responses[i] = organizationResponse{
			ID:        organization.ID,
			Name:      organization.Name,
			Slug:      organization.Slug,
			CreatedAt: organization.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: organization.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	ctx.JSON(http.StatusOK, responses)
}
