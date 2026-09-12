package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"agronode/backend/internal/auth"
	"agronode/backend/internal/config"
	"agronode/backend/internal/models"
	"gorm.io/gorm"
)

func EnsureBootstrapAuthData(ctx context.Context, db *gorm.DB, cfg config.Config, logger *slog.Logger) error {
	var userCount int64
	if err := db.WithContext(ctx).Model(&models.User{}).Count(&userCount).Error; err != nil {
		return err
	}

	if userCount > 0 {
		return nil
	}

	organization := models.Organization{
		ID:   cfg.FrontendOrganizationID,
		Name: "AgroNode",
		Slug: "default",
	}
	if err := db.WithContext(ctx).FirstOrCreate(&organization, models.Organization{ID: cfg.FrontendOrganizationID}).Error; err != nil {
		return err
	}

	passwordHash, err := auth.HashPassword(cfg.FrontendLoginPassword)
	if err != nil {
		return err
	}

	user := models.User{
		OrganizationID: organization.ID,
		Email:          strings.ToLower(strings.TrimSpace(cfg.FrontendLoginEmail)),
		FullName:       "Admin",
		PasswordHash:   passwordHash,
		Role:           "admin",
	}
	if err := db.WithContext(ctx).Create(&user).Error; err != nil {
		return fmt.Errorf("create bootstrap user: %w", err)
	}

	if logger != nil {
		logger.Info("bootstrap auth user created", "email", user.Email, "organizationId", user.OrganizationID)
	}

	return nil
}
