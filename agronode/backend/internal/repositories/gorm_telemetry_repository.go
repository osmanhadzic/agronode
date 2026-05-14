package repositories

import (
	"context"
	"errors"

	"agronode/backend/internal/models"
	"gorm.io/gorm"
)

type GormTelemetryRepository struct {
	database *gorm.DB
}

func NewGormTelemetryRepository(database *gorm.DB) *GormTelemetryRepository {
	return &GormTelemetryRepository{database: database}
}

func (repository *GormTelemetryRepository) Save(context context.Context, reading models.TelemetryReading) error {
	entity := models.SensorData{
		DeviceID:    reading.DeviceID,
		Temperature: reading.Temperature,
		Humidity:    reading.Humidity,
		CreatedAt:   reading.CreatedAt,
	}

	return repository.database.WithContext(context).Create(&entity).Error
}

func (repository *GormTelemetryRepository) List(context context.Context) ([]models.TelemetryReading, error) {
	var entities []models.SensorData
	err := repository.database.WithContext(context).
		Order("created_at DESC").
		Find(&entities).Error
	if err != nil {
		return nil, err
	}

	return toReadings(entities), nil
}

func (repository *GormTelemetryRepository) ListByDeviceID(context context.Context, deviceID string) ([]models.TelemetryReading, error) {
	var entities []models.SensorData
	err := repository.database.WithContext(context).
		Where("device_id = ?", deviceID).
		Order("created_at DESC").
		Find(&entities).Error
	if err != nil {
		return nil, err
	}

	return toReadings(entities), nil
}

func (repository *GormTelemetryRepository) GetLatestByDeviceID(context context.Context, deviceID string) (models.TelemetryReading, error) {
	var entity models.SensorData
	err := repository.database.WithContext(context).
		Where("device_id = ?", deviceID).
		Order("created_at DESC").
		First(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.TelemetryReading{}, ErrNotFound
		}

		return models.TelemetryReading{}, err
	}

	return models.TelemetryReading{
		DeviceID:    entity.DeviceID,
		Temperature: entity.Temperature,
		Humidity:    entity.Humidity,
		CreatedAt:   entity.CreatedAt,
	}, nil
}

func toReadings(entities []models.SensorData) []models.TelemetryReading {
	readings := make([]models.TelemetryReading, 0, len(entities))
	for _, entity := range entities {
		readings = append(readings, models.TelemetryReading{
			DeviceID:    entity.DeviceID,
			Temperature: entity.Temperature,
			Humidity:    entity.Humidity,
			CreatedAt:   entity.CreatedAt,
		})
	}

	return readings
}
