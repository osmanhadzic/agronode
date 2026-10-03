package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"agronode/backend/internal/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormTriggerRepository struct {
	database *gorm.DB
}

func NewGormTriggerRepository(database *gorm.DB) *GormTriggerRepository {
	return &GormTriggerRepository{database: database}
}

func (repository *GormTriggerRepository) Upsert(context context.Context, deviceID, sensorID string, trigger models.SensorTrigger) error {
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
		SensorID:       sensorID,
		MinValue:       trigger.Min,
		MaxValue:       trigger.Max,
		TargetDeviceID: targetDeviceID,
		UpdatedAt:      time.Now().UTC(),
	}
	if len(trigger.FuzzyConfig) > 0 {
		entity.FuzzyConfig = datatypes.JSON(trigger.FuzzyConfig)
	}

	m := map[string]interface{}{
		"min_value":        entity.MinValue,
		"max_value":        entity.MaxValue,
		"target_device_id": entity.TargetDeviceID,
		"updated_at":       entity.UpdatedAt,
	}
	if len(trigger.FuzzyConfig) > 0 {
		m["fuzzy_config"] = entity.FuzzyConfig
	}

	db := repository.database.WithContext(context).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "device_id"},
				{Name: "sensor_id"},
			},
			DoUpdates: clause.Assignments(m),
		})

	if len(trigger.FuzzyConfig) > 0 {
		return db.Create(&entity).Error
	}

	return db.Omit("fuzzy_config").Create(&entity).Error
}

func (repository *GormTriggerRepository) GetByDeviceAndSensor(context context.Context, deviceID, sensorID string) (models.SensorTrigger, error) {
	var entity models.SensorTriggerEntity
	db := repository.database.WithContext(context).
		Select("sensor_triggers.id, sensor_triggers.device_id, sensor_triggers.sensor_id, sensor_triggers.min_value, sensor_triggers.max_value, sensor_triggers.target_device_id, sensor_triggers.updated_at").
		Where("sensor_triggers.device_id = ? AND sensor_triggers.sensor_id = ?", deviceID, sensorID)

	if organizationID, ok := organizationIDFromContext(context); ok {
		db = db.
			Joins("JOIN devices ON devices.device_id = sensor_triggers.device_id").
			Where("devices.organization_id = ?", organizationID)
	}

	err := db.First(&entity).Error
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
		FuzzyConfig:    json.RawMessage(entity.FuzzyConfig),
	}, nil
}

func (repository *GormTriggerRepository) ListByDeviceID(context context.Context, deviceID string) (map[string]models.SensorTrigger, error) {
	var entities []models.SensorTriggerEntity
	db := repository.database.WithContext(context).
		Select("sensor_triggers.id, sensor_triggers.device_id, sensor_triggers.sensor_id, sensor_triggers.min_value, sensor_triggers.max_value, sensor_triggers.target_device_id, sensor_triggers.updated_at").
		Where("sensor_triggers.device_id = ?", deviceID)

	if organizationID, ok := organizationIDFromContext(context); ok {
		db = db.
			Joins("JOIN devices ON devices.device_id = sensor_triggers.device_id").
			Where("devices.organization_id = ?", organizationID)
	}

	err := db.Find(&entities).Error
	if err != nil {
		return nil, err
	}

	triggers := make(map[string]models.SensorTrigger, len(entities))
	for _, entity := range entities {
		triggers[entity.SensorID] = models.SensorTrigger{
			Min:            entity.MinValue,
			Max:            entity.MaxValue,
			TargetDeviceID: valueOrEmpty(entity.TargetDeviceID),
			FuzzyConfig:    json.RawMessage(entity.FuzzyConfig),
		}
	}

	return triggers, nil
}

func (repository *GormTriggerRepository) DeleteByDeviceAndSensor(context context.Context, deviceID, sensorID string) error {
	db := repository.database.WithContext(context).
		Where("sensor_triggers.device_id = ? AND sensor_triggers.sensor_id = ?", deviceID, sensorID)

	if organizationID, ok := organizationIDFromContext(context); ok {
		scopedDevices := repository.database.WithContext(context).
			Model(&models.Device{}).
			Select("device_id").
			Where("organization_id = ?", organizationID)

		db = db.Where("sensor_triggers.device_id IN (?)", scopedDevices)
	}

	result := db.Delete(&models.SensorTriggerEntity{})
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
