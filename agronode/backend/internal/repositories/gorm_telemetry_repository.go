package repositories

import (
	"context"
	"time"

	"agronode/backend/internal/models"
	"gorm.io/gorm"
)

// GormTelemetryRepository reads devices from the relational database and
// stores sensor readings in the time-series database.
type GormTelemetryRepository struct {
	database *gorm.DB
	series   *SensorSeriesStore
}

func NewGormTelemetryRepository(database, timeseries *gorm.DB) *GormTelemetryRepository {
	return &GormTelemetryRepository{database: database, series: NewSensorSeriesStore(timeseries)}
}

// deviceVisible reports whether the device belongs to the organization in the context (always true without one).
func (repository *GormTelemetryRepository) deviceVisible(context context.Context, deviceID string) (bool, error) {
	organizationID, ok := organizationIDFromContext(context)
	if !ok {
		return true, nil
	}

	var count int64
	err := repository.database.WithContext(context).
		Model(&models.Device{}).
		Where("device_id = ? AND organization_id = ?", deviceID, organizationID).
		Count(&count).Error

	return count > 0, err
}

func (repository *GormTelemetryRepository) Save(context context.Context, reading models.TelemetryReading) error {
	if organizationID, ok := organizationIDFromContext(context); ok {
		var count int64
		if err := repository.database.WithContext(context).
			Model(&models.Device{}).
			Where("device_id = ? AND organization_id = ?", reading.DeviceID, organizationID).
			Count(&count).Error; err != nil {
			return err
		}

		if count == 0 {
			return ErrDeviceNotFound
		}
	}

	for sensorID, value := range normalizeSensors(reading) {
		if err := repository.series.Insert(context, sensorID, reading.DeviceID, value, reading.CreatedAt); err != nil {
			return err
		}
	}

	return nil
}

func (repository *GormTelemetryRepository) List(context context.Context) ([]models.TelemetryReading, error) {
	query := SeriesQuery{}

	if organizationID, ok := organizationIDFromContext(context); ok {
		var deviceIDs []string
		if err := repository.database.WithContext(context).
			Model(&models.Device{}).
			Where("organization_id = ?", organizationID).
			Pluck("device_id", &deviceIDs).Error; err != nil {
			return nil, err
		}
		if len(deviceIDs) == 0 {
			return []models.TelemetryReading{}, nil
		}

		query.DeviceIDs = deviceIDs
	}

	return repository.readings(context, query)
}

func (repository *GormTelemetryRepository) ListByDeviceID(context context.Context, deviceID string) ([]models.TelemetryReading, error) {
	return repository.listForDevice(context, SeriesQuery{DeviceIDs: []string{deviceID}})
}

func (repository *GormTelemetryRepository) ListByDeviceIDAndSensorID(context context.Context, deviceID, sensorID string) ([]models.TelemetryReading, error) {
	return repository.listForDevice(context, SeriesQuery{DeviceIDs: []string{deviceID}, SensorID: sensorID})
}

func (repository *GormTelemetryRepository) ListByDeviceIDWithDateRange(context context.Context, deviceID string, dateRange DateRange) ([]models.TelemetryReading, error) {
	return repository.listForDevice(context, SeriesQuery{DeviceIDs: []string{deviceID}, Start: dateRange.StartDate, End: dateRange.EndDate})
}

func (repository *GormTelemetryRepository) listForDevice(context context.Context, query SeriesQuery) ([]models.TelemetryReading, error) {
	visible, err := repository.deviceVisible(context, query.DeviceIDs[0])
	if err != nil {
		return nil, err
	}
	if !visible {
		return []models.TelemetryReading{}, nil
	}

	return repository.readings(context, query)
}

// GetLatestByDeviceID merges the newest value of every sensor of the device.
func (repository *GormTelemetryRepository) GetLatestByDeviceID(context context.Context, deviceID string) (models.TelemetryReading, error) {
	visible, err := repository.deviceVisible(context, deviceID)
	if err != nil {
		return models.TelemetryReading{}, err
	}
	if !visible {
		return models.TelemetryReading{}, ErrNotFound
	}

	points, err := repository.series.Query(context, SeriesQuery{DeviceIDs: []string{deviceID}, LatestPerSensor: true})
	if err != nil {
		return models.TelemetryReading{}, err
	}
	if len(points) == 0 {
		return models.TelemetryReading{}, ErrNotFound
	}

	sensors := make(map[string]float64, len(points))
	for _, point := range points {
		sensors[point.SensorID] = point.Value
	}

	return newReading(deviceID, "", sensors, points[0].Time), nil
}

func (repository *GormTelemetryRepository) readings(context context.Context, query SeriesQuery) ([]models.TelemetryReading, error) {
	points, err := repository.series.Query(context, query)
	if err != nil {
		return nil, err
	}

	readings := make([]models.TelemetryReading, 0, len(points))
	for _, point := range points {
		readings = append(readings, newReading(point.DeviceID, point.SensorID, map[string]float64{point.SensorID: point.Value}, point.Time))
	}

	return readings, nil
}

func newReading(deviceID, sensorID string, sensors map[string]float64, createdAt time.Time) models.TelemetryReading {
	temperature, humidity := resolveSensorColumns(models.TelemetryReading{}, sensors)

	return models.TelemetryReading{
		DeviceID:    deviceID,
		SensorID:    sensorID,
		Temperature: temperature,
		Humidity:    humidity,
		Sensors:     sensors,
		CreatedAt:   createdAt,
	}
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

func resolveSensorColumns(reading models.TelemetryReading, sensors map[string]float64) (float64, float64) {
	temperature := reading.Temperature
	humidity := reading.Humidity

	if value, ok := sensors["temperature"]; ok {
		temperature = value
	}
	if value, ok := sensors["dht11-temp"]; ok {
		temperature = value
	}

	if value, ok := sensors["humidity"]; ok {
		humidity = value
	}
	if value, ok := sensors["humidity_dht11"]; ok {
		humidity = value
	}
	if value, ok := sensors["dht11-humidity"]; ok {
		humidity = value
	}

	return temperature, humidity
}
