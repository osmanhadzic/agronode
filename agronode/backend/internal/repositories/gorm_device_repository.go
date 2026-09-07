package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"agronode/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormDeviceRepository struct {
	database *gorm.DB
}

func NewGormDeviceRepository(database *gorm.DB) *GormDeviceRepository {
	return &GormDeviceRepository{database: database}
}

// Create creates a new device or returns existing one (idempotent)
func (repository *GormDeviceRepository) Create(ctx context.Context, device *models.Device) error {
	// Use ON CONFLICT to make registration idempotent
	result := repository.database.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "device_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
		}).
		Create(device)

	if result.Error != nil {
		return result.Error
	}

	// If the device already existed, fetch it to populate the device pointer with correct data
	if result.RowsAffected == 0 {
		existingDevice, err := repository.GetByDeviceID(ctx, device.DeviceID)
		if err != nil {
			return err
		}
		*device = *existingDevice
	}

	return nil
}

// GetByDeviceID retrieves a device by its device_id
func (repository *GormDeviceRepository) GetByDeviceID(ctx context.Context, deviceID string) (*models.Device, error) {
	var device models.Device
	err := repository.database.WithContext(ctx).
		Where("device_id = ?", deviceID).
		First(&device).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}

	return &device, nil
}

// Update updates an existing device
func (repository *GormDeviceRepository) Update(ctx context.Context, device *models.Device) error {
	device.UpdatedAt = time.Now()
	result := repository.database.WithContext(ctx).
		Model(device).
		Updates(device)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrDeviceNotFound
	}

	return nil
}

// List retrieves devices with pagination, optional status filter and search
func (repository *GormDeviceRepository) List(ctx context.Context, query DeviceListQuery) ([]models.Device, error) {
	db := repository.database.WithContext(ctx).Model(&models.Device{})

	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	if query.Search != "" {
		likeTerm := "%" + strings.TrimSpace(query.Search) + "%"
		db = db.Where("device_id ILIKE ? OR firmware_version ILIKE ?", likeTerm, likeTerm)
	}

	if len(query.Tags) > 0 {
		tagsJSON, marshalErr := json.Marshal(query.Tags)
		if marshalErr != nil {
			return nil, marshalErr
		}
		db = db.Where("tags @> ?::jsonb", string(tagsJSON))
	}

	var devices []models.Device
	err := db.
		Order("created_at DESC").
		Offset((query.Page - 1) * query.Limit).
		Limit(query.Limit).
		Find(&devices).Error

	if err != nil {
		return nil, err
	}

	return devices, nil
}

// ListInactiveOnline retrieves online devices that haven't been seen since the given time
func (repository *GormDeviceRepository) ListInactiveOnline(ctx context.Context, inactiveSince time.Time) ([]models.Device, error) {
	var devices []models.Device
	err := repository.database.WithContext(ctx).
		Where("status = ? AND (last_seen IS NULL OR last_seen < ?)", models.DeviceStatusOnline, inactiveSince).
		Order("last_seen ASC").
		Find(&devices).Error

	if err != nil {
		return nil, err
	}

	return devices, nil
}
