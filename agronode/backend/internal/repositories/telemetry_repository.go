package repositories

import (
	"context"
	"errors"
	"time"

	"agronode/backend/internal/models"
)

var ErrNotFound = errors.New("telemetry not found")

type DateRange struct {
	StartDate *time.Time
	EndDate   *time.Time
}

type TelemetryRepository interface {
	Save(context context.Context, reading models.TelemetryReading) error
	List(context context.Context) ([]models.TelemetryReading, error)
	ListByDeviceID(context context.Context, deviceID string) ([]models.TelemetryReading, error)
	ListByDeviceIDWithDateRange(context context.Context, deviceID string, dateRange DateRange) ([]models.TelemetryReading, error)
	GetLatestByDeviceID(context context.Context, deviceID string) (models.TelemetryReading, error)
}
