package repositories

import (
	"context"
	"errors"
	"time"

	"agronode/backend/internal/models"
)

var ErrDeviceNotFound = errors.New("device not found")

type DeviceListQuery struct {
	Page   int
	Limit  int
	Status string
	Search string
	Tags   []string
}

type DeviceRepository interface {
	Create(ctx context.Context, device *models.Device) error
	GetByDeviceID(ctx context.Context, deviceID string) (*models.Device, error)
	Update(ctx context.Context, device *models.Device) error
	List(ctx context.Context, query DeviceListQuery) ([]models.Device, error)
	ListInactiveOnline(ctx context.Context, inactiveSince time.Time) ([]models.Device, error)
}
