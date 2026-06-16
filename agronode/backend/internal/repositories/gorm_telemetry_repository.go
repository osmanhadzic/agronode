package repositories

import (
	"context"
	"encoding/json"
	"errors"

	"agronode/backend/internal/models"
	"gorm.io/gorm/clause"
	"gorm.io/gorm"
)

type GormTelemetryRepository struct {
	database *gorm.DB
}

func NewGormTelemetryRepository(database *gorm.DB) *GormTelemetryRepository {
	return &GormTelemetryRepository{database: database}
}

func (repository *GormTelemetryRepository) Save(context context.Context, reading models.TelemetryReading) error {
	sensors := normalizeSensors(reading)
	sensorsJSON, marshalError := json.Marshal(sensors)
	if marshalError != nil {
		return marshalError
	}

	metaJSON, marshalError := json.Marshal(reading.Meta)
	if marshalError != nil {
		return marshalError
	}

	entity := models.SensorData{
		DeviceID:    reading.DeviceID,
		Temperature: reading.Temperature,
		Humidity:    reading.Humidity,
		Sensors:     string(sensorsJSON),
		Meta:        string(metaJSON),
		CreatedAt:   reading.CreatedAt,
	}

	return repository.database.WithContext(context).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "device_id"},
				{Name: "created_at"},
				{Name: "sensors"},
			},
			DoNothing: true,
		}).
		Create(&entity).Error
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

func (repository *GormTelemetryRepository) ListByDeviceIDWithDateRange(context context.Context, deviceID string, dateRange DateRange) ([]models.TelemetryReading, error) {
	var entities []models.SensorData
	query := repository.database.WithContext(context).Where("device_id = ?", deviceID)

	if dateRange.StartDate != nil {
		query = query.Where("created_at >= ?", *dateRange.StartDate)
	}

	if dateRange.EndDate != nil {
		query = query.Where("created_at <= ?", *dateRange.EndDate)
	}

	err := query.Order("created_at DESC").Find(&entities).Error
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

	sensors := parseSensors(entity.Sensors)

	temperature := entity.Temperature
	if sensorTemperature, hasTemperature := sensors["temperature"]; hasTemperature {
		temperature = sensorTemperature
	}

	humidity := entity.Humidity
	if sensorHumidity, hasHumidity := sensors["humidity"]; hasHumidity {
		humidity = sensorHumidity
	}

	return models.TelemetryReading{
		DeviceID:    entity.DeviceID,
		Temperature: temperature,
		Humidity:    humidity,
		Sensors:     sensors,
		Meta:        parseMeta(entity.Meta),
		CreatedAt:   entity.CreatedAt,
	}, nil
}

func toReadings(entities []models.SensorData) []models.TelemetryReading {
	readings := make([]models.TelemetryReading, 0, len(entities))
	for _, entity := range entities {
		sensors := parseSensors(entity.Sensors)

		temperature := entity.Temperature
		if sensorTemperature, hasTemperature := sensors["temperature"]; hasTemperature {
			temperature = sensorTemperature
		}

		humidity := entity.Humidity
		if sensorHumidity, hasHumidity := sensors["humidity"]; hasHumidity {
			humidity = sensorHumidity
		}

		readings = append(readings, models.TelemetryReading{
			DeviceID:    entity.DeviceID,
			Temperature: temperature,
			Humidity:    humidity,
			Sensors:     sensors,
			Meta:        parseMeta(entity.Meta),
			CreatedAt:   entity.CreatedAt,
		})
	}

	return readings
}

func normalizeSensors(reading models.TelemetryReading) map[string]float64 {
	sensors := map[string]float64{}
	for key, value := range reading.Sensors {
		sensors[key] = value
	}

	if len(sensors) == 0 {
		sensors["temperature"] = reading.Temperature
		sensors["humidity"] = reading.Humidity
	}

	return sensors
}

func parseMeta(raw string) *models.DeviceMeta {
	if raw == "" || raw == "null" || raw == "{}" {
		return nil
	}

	var meta models.DeviceMeta
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil
	}

	if meta.Firmware == "" && meta.IP == "" && meta.RSSI == 0 && meta.Uptime == 0 {
		return nil
	}

	return &meta
}

func parseSensors(raw string) map[string]float64 {
	if raw == "" {
		return map[string]float64{}
	}

	var sensors map[string]float64
	if err := json.Unmarshal([]byte(raw), &sensors); err != nil {
		return map[string]float64{}
	}

	if sensors == nil {
		return map[string]float64{}
	}

	return sensors
}
