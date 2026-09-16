package repositories

import (
	"context"
	"errors"
	"time"

	"agronode/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormSensorRepository struct {
	database *gorm.DB
}

func NewGormSensorRepository(database *gorm.DB) *GormSensorRepository {
	return &GormSensorRepository{database: database}
}

func (repository *GormSensorRepository) Upsert(ctx context.Context, deviceID, sensorID string) error {
	entity := models.Sensor{
		DeviceID:  deviceID,
		SensorID:  sensorID,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	return repository.database.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "device_id"},
				{Name: "sensor_id"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"updated_at": entity.UpdatedAt,
			}),
		}).
		Create(&entity).Error
}

func (repository *GormSensorRepository) ListByDeviceID(ctx context.Context, deviceID string) ([]models.Sensor, error) {
	var sensors []models.Sensor
	db := repository.database.WithContext(ctx).
		Where("sensors.device_id = ?", deviceID)

	if organizationID, ok := organizationIDFromContext(ctx); ok {
		db = db.
			Joins("JOIN devices ON devices.device_id = sensors.device_id").
			Where("devices.organization_id = ?", organizationID)
	}

	if err := db.Order("sensors.sensor_id ASC").Find(&sensors).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Sensor{}, nil
		}

		return nil, err
	}

	return sensors, nil
}
