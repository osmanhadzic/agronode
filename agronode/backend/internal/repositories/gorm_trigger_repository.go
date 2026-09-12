package repositories

import (
	"context"
	"errors"
	"time"

	"agronode/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormTriggerRepository struct {
	database *gorm.DB
}

func NewGormTriggerRepository(database *gorm.DB) *GormTriggerRepository {
	return &GormTriggerRepository{database: database}
}

func (repository *GormTriggerRepository) Upsert(context context.Context, deviceID, sensor string, trigger models.SensorTrigger) error {
	if organizationID, ok := organizationIDFromContext(context); ok {
		var count int64
		if err := repository.database.WithContext(context).
			Model(&models.Device{}).
			Where("device_id = ? AND organization_id = ?", deviceID, organizationID).
			Count(&count).Error; err != nil {
			return err
		}

		if count == 0 {
			return ErrNotFound
		}
	}

	var targetDeviceID *string
	if trigger.TargetDeviceID != "" {
		targetDeviceID = &trigger.TargetDeviceID
	}

	entity := models.SensorTriggerEntity{
		DeviceID:       deviceID,
		Sensor:         sensor,
		MinValue:       trigger.Min,
		MaxValue:       trigger.Max,
		TargetDeviceID: targetDeviceID,
		UpdatedAt:      time.Now().UTC(),
	}

	return repository.database.WithContext(context).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "device_id"},
				{Name: "sensor"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"min_value":        entity.MinValue,
				"max_value":        entity.MaxValue,
				"target_device_id": entity.TargetDeviceID,
				"updated_at":       entity.UpdatedAt,
			}),
		}).
		Create(&entity).Error
}

func (repository *GormTriggerRepository) GetByDeviceAndSensor(context context.Context, deviceID, sensor string) (models.SensorTrigger, error) {
	var entity models.SensorTriggerEntity
	db := repository.database.WithContext(context).
		Where("sensor_triggers.device_id = ? AND sensor_triggers.sensor = ?", deviceID, sensor)

	if organizationID, ok := organizationIDFromContext(context); ok {
		db = db.
			Joins("JOIN devices ON devices.device_id = sensor_triggers.device_id").
			Where("devices.organization_id = ?", organizationID)
	}

	err := db.
		First(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.SensorTrigger{}, ErrNotFound
		}

		return models.SensorTrigger{}, err
	}

	return models.SensorTrigger{
		Min:            entity.MinValue,
		Max:            entity.MaxValue,
		TargetDeviceID: valueOrEmpty(entity.TargetDeviceID),
	}, nil
}

func (repository *GormTriggerRepository) ListByDeviceID(context context.Context, deviceID string) (map[string]models.SensorTrigger, error) {
	var entities []models.SensorTriggerEntity
	db := repository.database.WithContext(context).
		Where("sensor_triggers.device_id = ?", deviceID)

	if organizationID, ok := organizationIDFromContext(context); ok {
		db = db.
			Joins("JOIN devices ON devices.device_id = sensor_triggers.device_id").
			Where("devices.organization_id = ?", organizationID)
	}

	err := db.
		Find(&entities).Error
	if err != nil {
		return nil, err
	}

	triggers := make(map[string]models.SensorTrigger, len(entities))
	for _, entity := range entities {
		triggers[entity.Sensor] = models.SensorTrigger{
			Min:            entity.MinValue,
			Max:            entity.MaxValue,
			TargetDeviceID: valueOrEmpty(entity.TargetDeviceID),
		}
	}

	return triggers, nil
}

func (repository *GormTriggerRepository) DeleteByDeviceAndSensor(context context.Context, deviceID, sensor string) error {
	db := repository.database.WithContext(context).
		Where("sensor_triggers.device_id = ? AND sensor_triggers.sensor = ?", deviceID, sensor)

	if organizationID, ok := organizationIDFromContext(context); ok {
		scopedDevices := repository.database.WithContext(context).
			Model(&models.Device{}).
			Select("device_id").
			Where("organization_id = ?", organizationID)

		db = db.Where("sensor_triggers.device_id IN (?)", scopedDevices)
	}

	result := db.
		Delete(&models.SensorTriggerEntity{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
