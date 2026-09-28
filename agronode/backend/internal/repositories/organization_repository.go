package repositories

import (
	"context"

	"agronode/backend/internal/models"
)

type OrganizationRepository interface {
	List(ctx context.Context) ([]models.Organization, error)
}
