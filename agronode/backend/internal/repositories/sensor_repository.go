package repositories

import (
	"context"

	"agronode/backend/internal/models"
)

type SensorRepository interface {
	Upsert(context context.Context, deviceID, sensorID string) error
	ListByDeviceID(context context.Context, deviceID string) ([]models.Sensor, error)
}
