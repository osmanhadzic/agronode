package repositories

import (
	"context"
	"errors"
	"time"

	"agronode/backend/internal/models"
	"gorm.io/gorm"
)

var (
	// ErrProvisioningTokenInvalid covers unknown, expired, revoked and already consumed tokens.
	ErrProvisioningTokenInvalid = errors.New("provisioning token is invalid or expired")
)

// ProvisionedDevice is the data written for a newly registered device.
type ProvisionedDevice struct {
	DeviceID         string
	HardwareID       string
	MQTTPasswordHash string
}

type ProvisioningRepository interface {
	CreateToken(ctx context.Context, token *models.ProvisioningToken) error
	AssetUnitBelongsToOrganization(ctx context.Context, assetUnitID uint, organizationID uint) (bool, error)
	// RegisterDevice atomically consumes the token and creates the device; any failure leaves the token unconsumed.
	RegisterDevice(ctx context.Context, tokenHash string, now time.Time, device ProvisionedDevice) (*models.Device, error)
	GetByMQTTUsername(ctx context.Context, username string) (*models.Device, error)
	SetCredentials(ctx context.Context, deviceID string, passwordHash string, now time.Time) error
	SetLifecycleStatus(ctx context.Context, deviceID string, status string) error
}

type GormProvisioningRepository struct {
	database *gorm.DB
}

func NewGormProvisioningRepository(database *gorm.DB) *GormProvisioningRepository {
	return &GormProvisioningRepository{database: database}
}

func (repository *GormProvisioningRepository) CreateToken(ctx context.Context, token *models.ProvisioningToken) error {
	return repository.database.WithContext(ctx).Create(token).Error
}

func (repository *GormProvisioningRepository) AssetUnitBelongsToOrganization(ctx context.Context, assetUnitID uint, organizationID uint) (bool, error) {
	var count int64
	err := repository.database.WithContext(ctx).Model(&models.Zone{}).
		Where("id = ? AND organization_id = ?", assetUnitID, organizationID).
		Count(&count).Error
	return count > 0, err
}

func (repository *GormProvisioningRepository) RegisterDevice(ctx context.Context, tokenHash string, now time.Time, provisioned ProvisionedDevice) (*models.Device, error) {
	var created *models.Device

	err := repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// A single conditional UPDATE is the atomic gate: only one concurrent caller can match.
		consume := tx.Model(&models.ProvisioningToken{}).
			Where("token_hash = ? AND consumed_at IS NULL AND revoked_at IS NULL AND expires_at > ?", tokenHash, now).
			Updates(map[string]any{"consumed_at": now, "consumed_device_id": provisioned.DeviceID})
		if consume.Error != nil {
			return consume.Error
		}
		if consume.RowsAffected != 1 {
			return ErrProvisioningTokenInvalid
		}

		var token models.ProvisioningToken
		if err := tx.Where("token_hash = ?", tokenHash).First(&token).Error; err != nil {
			return err
		}

		username := provisioned.DeviceID
		hardwareID := provisioned.HardwareID
		device := &models.Device{
			DeviceID:          provisioned.DeviceID,
			HardwareID:        &hardwareID,
			OrganizationID:    &token.OrganizationID,
			ZoneID:            token.AssetUnitID,
			DeviceType:        models.DeviceTypePublisher,
			Status:            models.DeviceStatusUnknown,
			Tags:              []string{},
			DesiredState:      models.DeviceShadowState{},
			ReportedState:     models.DeviceShadowState{},
			DiscoveredSensors: []string{},
			MQTTUsername:      &username,
			MQTTPasswordHash:  provisioned.MQTTPasswordHash,
			LifecycleStatus:   models.LifecycleActive,
			RegisteredAt:      &now,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		if err := tx.Create(device).Error; err != nil {
			return translateHardwareIDError(err)
		}

		created = device
		return nil
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

// GetByMQTTUsername is deliberately unscoped: the broker has no tenant context, and ownership is read from the row.
func (repository *GormProvisioningRepository) GetByMQTTUsername(ctx context.Context, username string) (*models.Device, error) {
	var device models.Device
	if err := repository.database.WithContext(ctx).Where("mqtt_username = ?", username).First(&device).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}

	return &device, nil
}

func (repository *GormProvisioningRepository) SetCredentials(ctx context.Context, deviceID string, passwordHash string, now time.Time) error {
	db := repository.database.WithContext(ctx).Model(&models.Device{}).
		Where("device_id = ? AND lifecycle_status IN ?", deviceID, []string{models.LifecyclePending, models.LifecycleActive})
	if organizationID, ok := organizationIDFromContext(ctx); ok {
		db = db.Where("organization_id = ?", organizationID)
	}

	result := db.Updates(map[string]any{
		"mqtt_username":          deviceID,
		"mqtt_password_hash":     passwordHash,
		"lifecycle_status":       models.LifecycleActive,
		"credentials_rotated_at": now,
		"updated_at":             now,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrDeviceNotFound
	}

	return nil
}

func (repository *GormProvisioningRepository) SetLifecycleStatus(ctx context.Context, deviceID string, status string) error {
	db := repository.database.WithContext(ctx).Model(&models.Device{}).Where("device_id = ?", deviceID)
	if organizationID, ok := organizationIDFromContext(ctx); ok {
		db = db.Where("organization_id = ?", organizationID)
	}

	// Revocation is terminal; a revoked device can only be replaced by a new registration.
	db = db.Where("lifecycle_status <> ?", models.LifecycleRevoked)
	if status == models.LifecycleActive {
		db = db.Where("mqtt_password_hash IS NOT NULL AND mqtt_password_hash <> ''")
	}

	result := db.Updates(map[string]any{"lifecycle_status": status, "updated_at": time.Now()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrDeviceNotFound
	}

	return nil
}
