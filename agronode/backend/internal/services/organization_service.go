package services

import (
	"context"

	"agronode/backend/internal/models"
	"agronode/backend/internal/repositories"
)

type OrganizationService struct {
	repository repositories.OrganizationRepository
}

func NewOrganizationService(repository repositories.OrganizationRepository) *OrganizationService {
	return &OrganizationService{repository: repository}
}

func (service *OrganizationService) ListOrganizations(ctx context.Context) ([]models.Organization, error) {
	return service.repository.List(ctx)
}
