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
	entity := models.SensorTriggerEntity{
		DeviceID:  deviceID,
		Sensor:    sensor,
		MinValue:  trigger.Min,
		MaxValue:  trigger.Max,
		UpdatedAt: time.Now().UTC(),
	}

	return repository.database.WithContext(context).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "device_id"},
				{Name: "sensor"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"min_value":  entity.MinValue,
				"max_value":  entity.MaxValue,
				"updated_at": entity.UpdatedAt,
			}),
		}).
		Create(&entity).Error
}

func (repository *GormTriggerRepository) GetByDeviceAndSensor(context context.Context, deviceID, sensor string) (models.SensorTrigger, error) {
	var entity models.SensorTriggerEntity
	err := repository.database.WithContext(context).
		Where("device_id = ? AND sensor = ?", deviceID, sensor).
		First(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.SensorTrigger{}, ErrNotFound
		}

		return models.SensorTrigger{}, err
	}

	return models.SensorTrigger{
		Min: entity.MinValue,
		Max: entity.MaxValue,
	}, nil
}

func (repository *GormTriggerRepository) ListByDeviceID(context context.Context, deviceID string) (map[string]models.SensorTrigger, error) {
	var entities []models.SensorTriggerEntity
	err := repository.database.WithContext(context).
		Where("device_id = ?", deviceID).
		Find(&entities).Error
	if err != nil {
		return nil, err
	}

	triggers := make(map[string]models.SensorTrigger, len(entities))
	for _, entity := range entities {
		triggers[entity.Sensor] = models.SensorTrigger{
			Min: entity.MinValue,
			Max: entity.MaxValue,
		}
	}

	return triggers, nil
}

func (repository *GormTriggerRepository) DeleteByDeviceAndSensor(context context.Context, deviceID, sensor string) error {
	result := repository.database.WithContext(context).
		Where("device_id = ? AND sensor = ?", deviceID, sensor).
		Delete(&models.SensorTriggerEntity{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
