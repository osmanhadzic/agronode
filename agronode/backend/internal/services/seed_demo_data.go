package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"agronode/backend/internal/auth"
	"agronode/backend/internal/config"
	"agronode/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func EnsureDemoSeedData(ctx context.Context, db *gorm.DB, cfg config.Config, logger *slog.Logger) error {
	if !cfg.SeedDemoData {
		return nil
	}

	organization, err := upsertOrganization(ctx, db, models.Organization{
		ID:   cfg.FrontendOrganizationID,
		Name: "AgroNode Demo",
		Slug: "demo",
	})
	if err != nil {
		return err
	}

	adminPasswordHash, err := auth.HashPassword(cfg.FrontendLoginPassword)
	if err != nil {
		return err
	}

	if err := upsertUser(ctx, db, models.User{
		OrganizationID: organization.ID,
		Email:          cfg.FrontendLoginEmail,
		FullName:       "Admin User",
		PasswordHash:   adminPasswordHash,
		Role:           "admin",
	}); err != nil {
		return err
	}

	operatorPasswordHash, err := auth.HashPassword("operator123")
	if err != nil {
		return err
	}

	if err := upsertUser(ctx, db, models.User{
		OrganizationID: organization.ID,
		Email:          "operator@agronode.local",
		FullName:       "Field Operator",
		PasswordHash:   operatorPasswordHash,
		Role:           "user",
	}); err != nil {
		return err
	}

	northFarm, err := upsertFarm(ctx, db, models.Farm{
		OrganizationID: organization.ID,
		Name:           "North Farm",
	})
	if err != nil {
		return err
	}

	southFarm, err := upsertFarm(ctx, db, models.Farm{
		OrganizationID: organization.ID,
		Name:           "South Farm",
	})
	if err != nil {
		return err
	}

	northFieldA, err := upsertField(ctx, db, models.Field{
		OrganizationID: organization.ID,
		FarmID:         uintPtr(northFarm.ID),
		Name:           "North Field A",
	})
	if err != nil {
		return err
	}

	northFieldB, err := upsertField(ctx, db, models.Field{
		OrganizationID: organization.ID,
		FarmID:         uintPtr(northFarm.ID),
		Name:           "North Field B",
	})
	if err != nil {
		return err
	}

	southFieldA, err := upsertField(ctx, db, models.Field{
		OrganizationID: organization.ID,
		FarmID:         uintPtr(southFarm.ID),
		Name:           "South Field A",
	})
	if err != nil {
		return err
	}

	northZone1, err := upsertZone(ctx, db, models.Zone{
		OrganizationID: organization.ID,
		FieldID:        northFieldA.ID,
		Name:           "North Zone 1",
	})
	if err != nil {
		return err
	}

	northZone2, err := upsertZone(ctx, db, models.Zone{
		OrganizationID: organization.ID,
		FieldID:        northFieldA.ID,
		Name:           "North Zone 2",
	})
	if err != nil {
		return err
	}

	northFieldBZone1, err := upsertZone(ctx, db, models.Zone{
		OrganizationID: organization.ID,
		FieldID:        northFieldB.ID,
		Name:           "North Field B Zone",
	})
	if err != nil {
		return err
	}

	southZone1, err := upsertZone(ctx, db, models.Zone{
		OrganizationID: organization.ID,
		FieldID:        southFieldA.ID,
		Name:           "South Zone 1",
	})
	if err != nil {
		return err
	}

	if err := upsertDevice(ctx, db, models.Device{
		DeviceID:              "demo-device-1",
		OrganizationID:        uintPtr(organization.ID),
		ZoneID:                uintPtr(northZone1.ID),
		Status:                models.DeviceStatusOnline,
		FirmwareVersion:       "v1.0.0",
		Metadata:              models.DeviceMetadata{Hardware: map[string]string{"model": "ESP32", "location": "north-gateway"}},
		Tags:                  []string{"demo", "north"},
		DesiredState:          models.DeviceShadowState{"relay": "off"},
		ReportedState:         models.DeviceShadowState{"relay": "off"},
		DiscoveredSensors:     []string{"temperature", "humidity"},
		APIKeyHash:            hashDeviceSecret("demo-device-1-api-key"),
		ProvisioningTokenHash: hashDeviceSecret("demo-device-1-provisioning"),
		CreatedAt:             time.Now().UTC(),
		UpdatedAt:             time.Now().UTC(),
	}); err != nil {
		return err
	}

	if err := upsertDevice(ctx, db, models.Device{
		DeviceID:              "demo-device-2",
		OrganizationID:        uintPtr(organization.ID),
		ZoneID:                uintPtr(southZone1.ID),
		Status:                models.DeviceStatusUnknown,
		FirmwareVersion:       "v1.0.0",
		Metadata:              models.DeviceMetadata{Hardware: map[string]string{"model": "ESP32", "location": "south-node"}},
		Tags:                  []string{"demo", "south"},
		DesiredState:          models.DeviceShadowState{"relay": "auto"},
		ReportedState:         models.DeviceShadowState{"relay": "auto"},
		DiscoveredSensors:     []string{"temperature", "soil_moisture"},
		APIKeyHash:            hashDeviceSecret("demo-device-2-api-key"),
		ProvisioningTokenHash: hashDeviceSecret("demo-device-2-provisioning"),
		CreatedAt:             time.Now().UTC(),
		UpdatedAt:             time.Now().UTC(),
	}); err != nil {
		return err
	}

	if err := upsertDevice(ctx, db, models.Device{
		DeviceID:              "demo-device-3",
		OrganizationID:        uintPtr(organization.ID),
		ZoneID:                uintPtr(northZone2.ID),
		Status:                models.DeviceStatusOffline,
		FirmwareVersion:       "v1.0.0",
		Metadata:              models.DeviceMetadata{Hardware: map[string]string{"model": "ESP32", "location": "north-canopy"}},
		Tags:                  []string{"demo", "north", "canopy"},
		DesiredState:          models.DeviceShadowState{"relay": "off"},
		ReportedState:         models.DeviceShadowState{"relay": "off"},
		DiscoveredSensors:     []string{"temperature", "light"},
		APIKeyHash:            hashDeviceSecret("demo-device-3-api-key"),
		ProvisioningTokenHash: hashDeviceSecret("demo-device-3-provisioning"),
		CreatedAt:             time.Now().UTC(),
		UpdatedAt:             time.Now().UTC(),
	}); err != nil {
		return err
	}

	if err := upsertDevice(ctx, db, models.Device{
		DeviceID:              "demo-device-4",
		OrganizationID:        uintPtr(organization.ID),
		ZoneID:                uintPtr(northFieldBZone1.ID),
		Status:                models.DeviceStatusUnknown,
		FirmwareVersion:       "v1.0.0",
		Metadata:              models.DeviceMetadata{Hardware: map[string]string{"model": "ESP32", "location": "north-storage"}},
		Tags:                  []string{"demo", "north", "storage"},
		DesiredState:          models.DeviceShadowState{"relay": "auto"},
		ReportedState:         models.DeviceShadowState{"relay": "auto"},
		DiscoveredSensors:     []string{"temperature", "humidity", "co2"},
		APIKeyHash:            hashDeviceSecret("demo-device-4-api-key"),
		ProvisioningTokenHash: hashDeviceSecret("demo-device-4-provisioning"),
		CreatedAt:             time.Now().UTC(),
		UpdatedAt:             time.Now().UTC(),
	}); err != nil {
		return err
	}

	if logger != nil {
		logger.Info("demo seed data ensured", "organizationId", organization.ID, "farms", 2, "fields", 3, "zones", 4, "devices", 4, "users", 2)
	}

	return nil
}

func upsertOrganization(ctx context.Context, db *gorm.DB, organization models.Organization) (*models.Organization, error) {
	result := db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "slug", "updated_at"}),
		}).
		Create(&organization)
	if result.Error != nil {
		return nil, fmt.Errorf("seed organization: %w", result.Error)
	}

	return &organization, nil
}

func upsertFarm(ctx context.Context, db *gorm.DB, farm models.Farm) (*models.Farm, error) {
	result := db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "organization_id"}, {Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
		}).
		Create(&farm)
	if result.Error != nil {
		return nil, fmt.Errorf("seed farm: %w", result.Error)
	}

	return &farm, nil
}

func upsertField(ctx context.Context, db *gorm.DB, field models.Field) (*models.Field, error) {
	result := db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "organization_id"}, {Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"farm_id", "updated_at"}),
		}).
		Create(&field)
	if result.Error != nil {
		return nil, fmt.Errorf("seed field: %w", result.Error)
	}

	return &field, nil
}

func upsertZone(ctx context.Context, db *gorm.DB, zone models.Zone) (*models.Zone, error) {
	result := db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "field_id"}, {Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"organization_id", "updated_at"}),
		}).
		Create(&zone)
	if result.Error != nil {
		return nil, fmt.Errorf("seed zone: %w", result.Error)
	}

	return &zone, nil
}

func upsertUser(ctx context.Context, db *gorm.DB, user models.User) error {
	result := db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "organization_id"}, {Name: "email"}},
			DoUpdates: clause.AssignmentColumns([]string{"full_name", "password_hash", "role", "updated_at"}),
		}).
		Create(&user)
	if result.Error != nil {
		return fmt.Errorf("seed user: %w", result.Error)
	}

	return nil
}

func upsertDevice(ctx context.Context, db *gorm.DB, device models.Device) error {
	result := db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "device_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"organization_id",
				"zone_id",
				"status",
				"firmware_version",
				"metadata",
				"tags",
				"desired_state",
				"reported_state",
				"discovered_sensors",
				"api_key_hash",
				"provisioning_token_hash",
				"updated_at",
			}),
		}).
		Create(&device)
	if result.Error != nil {
		return fmt.Errorf("seed device: %w", result.Error)
	}

	return nil
}

func uintPtr(value uint) *uint {
	return &value
}
