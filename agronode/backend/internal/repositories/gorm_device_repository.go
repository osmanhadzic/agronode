package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"agronode/backend/internal/models"
	"github.com/jackc/pgx/v5/pgconn"
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
	// Use ON CONFLICT to make registration idempotent. A hardware ID is only
	// filled in when the existing row has none; it is never overwritten.
	result := repository.database.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "device_id"}},
			DoUpdates: clause.Set{
				{Column: clause.Column{Name: "updated_at"}, Value: gorm.Expr("excluded.updated_at")},
				{Column: clause.Column{Name: "hardware_id"}, Value: gorm.Expr("COALESCE(devices.hardware_id, excluded.hardware_id)")},
			},
		}).
		Create(device)

	if result.Error != nil {
		return translateHardwareIDError(result.Error)
	}

	if device.HardwareID == nil {
		return nil
	}

	// A concurrent registration of the same device_id is resolved by the upsert
	// above, so confirm what was actually stored instead of trusting the request.
	var stored *string
	if err := repository.database.WithContext(ctx).
		Model(&models.Device{}).
		Where("device_id = ?", device.DeviceID).
		Select("hardware_id").
		Scan(&stored).Error; err != nil {
		return err
	}

	if stored != nil && *stored != *device.HardwareID {
		return ErrHardwareIDMismatch
	}

	return nil
}

// GetByDeviceID retrieves a device by its device_id
func (repository *GormDeviceRepository) GetByDeviceID(ctx context.Context, deviceID string) (*models.Device, error) {
	var device models.Device
	db := repository.database.WithContext(ctx).
		Where("device_id = ?", deviceID)

	if organizationID, ok := organizationIDFromContext(ctx); ok {
		db = db.Where("organization_id = ?", organizationID)
	}

	if err := db.
		First(&device).Error; err != nil {
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
		return translateHardwareIDError(result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrDeviceNotFound
	}

	return nil
}

// List retrieves devices with pagination, optional status filter and search
func (repository *GormDeviceRepository) List(ctx context.Context, query DeviceListQuery) ([]models.Device, error) {
	db := repository.database.WithContext(ctx).Model(&models.Device{})

	if organizationID, ok := organizationIDFromContext(ctx); ok {
		db = db.Where("organization_id = ?", organizationID)
	}

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
	db := repository.database.WithContext(ctx).
		Where("status = ? AND (last_seen IS NULL OR last_seen < ?)", models.DeviceStatusOnline, inactiveSince)

	if organizationID, ok := organizationIDFromContext(ctx); ok {
		db = db.Where("organization_id = ?", organizationID)
	}

	err := db.
		Order("last_seen ASC").
		Find(&devices).Error

	if err != nil {
		return nil, err
	}

	return devices, nil
}

// translateHardwareIDError maps a unique-constraint violation on hardware_id to ErrDuplicateHardwareID.
func translateHardwareIDError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "hardware_id") {
		return ErrDuplicateHardwareID
	}

	// SQLite (used in tests) reports "UNIQUE constraint failed: devices.hardware_id".
	if strings.Contains(err.Error(), "UNIQUE constraint failed: devices.hardware_id") {
		return ErrDuplicateHardwareID
	}

	return err
}
