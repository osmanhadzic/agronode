package repositories

import (
	"context"
	"errors"
	"strings"

	"agronode/backend/internal/models"
	"gorm.io/gorm"
)

type GormUserRepository struct {
	database *gorm.DB
}

func NewGormUserRepository(database *gorm.DB) *GormUserRepository {
	return &GormUserRepository{database: database}
}

func (repository *GormUserRepository) Create(ctx context.Context, user *models.User) error {
	return repository.database.WithContext(ctx).Create(user).Error
}

func (repository *GormUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := repository.database.WithContext(ctx).
		Where("LOWER(email) = ?", strings.ToLower(strings.TrimSpace(email))).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return &user, nil
}
