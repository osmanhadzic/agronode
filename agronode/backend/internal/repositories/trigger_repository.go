package repositories

import (
	"context"

	"agronode/backend/internal/models"
)

type TriggerRepository interface {
	Upsert(context context.Context, deviceID, sensor string, trigger models.SensorTrigger) error
	GetByDeviceAndSensor(context context.Context, deviceID, sensor string) (models.SensorTrigger, error)
	ListByDeviceID(context context.Context, deviceID string) (map[string]models.SensorTrigger, error)
	DeleteByDeviceAndSensor(context context.Context, deviceID, sensor string) error
}
