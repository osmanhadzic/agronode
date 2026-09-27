package repositories

import (
	"context"

	"agronode/backend/internal/models"
	"gorm.io/gorm"
)

type GormOrganizationRepository struct {
	database *gorm.DB
}

func NewGormOrganizationRepository(database *gorm.DB) *GormOrganizationRepository {
	return &GormOrganizationRepository{database: database}
}

func (repository *GormOrganizationRepository) List(ctx context.Context) ([]models.Organization, error) {
	query := repository.database.WithContext(ctx).Model(&models.Organization{})
	if organizationID, ok := organizationIDFromContext(ctx); ok {
		query = query.Where("id = ?", organizationID)
	}

	organizations := make([]models.Organization, 0)
	if err := query.Order("id ASC").Find(&organizations).Error; err != nil {
		return nil, err
	}

	return organizations, nil
}
