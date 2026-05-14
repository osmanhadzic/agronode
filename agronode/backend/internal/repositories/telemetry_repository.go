package repositories

import (
	"context"
	"errors"

	"agronode/backend/internal/models"
)

var ErrNotFound = errors.New("telemetry not found")

type TelemetryRepository interface {
	Save(context context.Context, reading models.TelemetryReading) error
	List(context context.Context) ([]models.TelemetryReading, error)
	ListByDeviceID(context context.Context, deviceID string) ([]models.TelemetryReading, error)
	GetLatestByDeviceID(context context.Context, deviceID string) (models.TelemetryReading, error)
}
