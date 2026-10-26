package repositories

import (
	"context"
	"errors"
	"testing"

	"agronode/backend/internal/models"
)

func storedHardwareID(t *testing.T, repository *GormDeviceRepository, deviceID string) *string {
	t.Helper()

	device, err := repository.GetByDeviceID(context.Background(), deviceID)
	if err != nil {
		t.Fatalf("get %s: %v", deviceID, err)
	}

	return device.HardwareID
}

func TestGormDeviceRepository_CreateNeverOverwritesHardwareID(t *testing.T) {
	repository := NewGormDeviceRepository(newTestSensorDB(t))
	ctx := context.Background()
	first, second := "AABBCCDDEEFF", "112233445566"

	if err := repository.Create(ctx, &models.Device{DeviceID: "node-1", HardwareID: &first}); err != nil {
		t.Fatalf("create: %v", err)
	}

	err := repository.Create(ctx, &models.Device{DeviceID: "node-1", HardwareID: &second})
	if !errors.Is(err, ErrHardwareIDMismatch) {
		t.Fatalf("expected ErrHardwareIDMismatch, got %v", err)
	}
	if stored := storedHardwareID(t, repository, "node-1"); stored == nil || *stored != first {
		t.Fatalf("stored hardware id must stay %s, got %v", first, stored)
	}

	same := "AABBCCDDEEFF"
	if err := repository.Create(ctx, &models.Device{DeviceID: "node-1", HardwareID: &same}); err != nil {
		t.Fatalf("re-registering with the same hardware id must succeed: %v", err)
	}
}

func TestGormDeviceRepository_CreateFillsMissingHardwareID(t *testing.T) {
	repository := NewGormDeviceRepository(newTestSensorDB(t))
	ctx := context.Background()

	if err := repository.Create(ctx, &models.Device{DeviceID: "node-1"}); err != nil {
		t.Fatalf("create without hardware id: %v", err)
	}

	hardwareID := "AABBCCDDEEFF"
	if err := repository.Create(ctx, &models.Device{DeviceID: "node-1", HardwareID: &hardwareID}); err != nil {
		t.Fatalf("create with hardware id: %v", err)
	}
	if stored := storedHardwareID(t, repository, "node-1"); stored == nil || *stored != hardwareID {
		t.Fatalf("expected hardware id to be filled in, got %v", stored)
	}

	if err := repository.Create(ctx, &models.Device{DeviceID: "node-1"}); err != nil {
		t.Fatalf("re-registering without hardware id must succeed: %v", err)
	}
	if stored := storedHardwareID(t, repository, "node-1"); stored == nil || *stored != hardwareID {
		t.Fatalf("hardware id must survive a registration without one, got %v", stored)
	}
}

func TestGormDeviceRepository_UpdateDuplicateHardwareID(t *testing.T) {
	db := newTestSensorDB(t)
	if err := db.Exec("CREATE UNIQUE INDEX idx_devices_hardware_id ON devices (hardware_id) WHERE hardware_id IS NOT NULL").Error; err != nil {
		t.Fatalf("create hardware_id index: %v", err)
	}
	repository := NewGormDeviceRepository(db)
	ctx := context.Background()
	taken := "AABBCCDDEEFF"

	if err := repository.Create(ctx, &models.Device{DeviceID: "node-1", HardwareID: &taken}); err != nil {
		t.Fatalf("create node-1: %v", err)
	}
	if err := repository.Create(ctx, &models.Device{DeviceID: "node-2"}); err != nil {
		t.Fatalf("create node-2: %v", err)
	}

	other := "AABBCCDDEEFF"
	err := repository.Update(ctx, &models.Device{ID: 2, DeviceID: "node-2", HardwareID: &other})
	if !errors.Is(err, ErrDuplicateHardwareID) {
		t.Fatalf("expected ErrDuplicateHardwareID from Update, got %v", err)
	}
}

func TestGormDeviceRepository_DuplicateHardwareID(t *testing.T) {
	db := newTestSensorDB(t)
	if err := db.Exec("CREATE UNIQUE INDEX idx_devices_hardware_id ON devices (hardware_id) WHERE hardware_id IS NOT NULL").Error; err != nil {
		t.Fatalf("create hardware_id index: %v", err)
	}
	repository := NewGormDeviceRepository(db)
	ctx := context.Background()
	hardwareID := "AABBCCDDEEFF"

	if err := repository.Create(ctx, &models.Device{DeviceID: "node-1", HardwareID: &hardwareID}); err != nil {
		t.Fatalf("create first device: %v", err)
	}

	duplicate := "AABBCCDDEEFF"
	err := repository.Create(ctx, &models.Device{DeviceID: "node-2", HardwareID: &duplicate})
	if !errors.Is(err, ErrDuplicateHardwareID) {
		t.Fatalf("expected ErrDuplicateHardwareID, got %v", err)
	}
}
