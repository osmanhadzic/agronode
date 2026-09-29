package repositories

import (
	"context"
	"fmt"
	"testing"
	"time"

	"agronode/backend/internal/models"
	"agronode/backend/internal/tenancy"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestTriggerDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := db.Exec(`
		CREATE TABLE devices (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id TEXT NOT NULL UNIQUE,
			organization_id INTEGER,
			asset_unit_id INTEGER,
			device_type TEXT NOT NULL DEFAULT 'publisher',
			status TEXT,
			registration_status TEXT NOT NULL DEFAULT 'active',
			firmware_version TEXT,
			provisioning_status TEXT,
			certificate_serial TEXT,
			certificate_expires_at DATETIME,
			metadata TEXT,
			tags TEXT,
			desired_state TEXT,
			reported_state TEXT,
			discovered_sensors TEXT,
			api_key_hash TEXT,
			provisioning_token_hash TEXT,
			last_seen DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		);
	`).Error; err != nil {
		t.Fatalf("create devices table: %v", err)
	}

	if err := db.Exec(`
		CREATE TABLE sensor_triggers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id TEXT NOT NULL,
			sensor_id TEXT NOT NULL,
			min_value REAL,
			max_value REAL,
			target_device_id TEXT,
			updated_at DATETIME NOT NULL,
			UNIQUE(device_id, sensor_id)
		);
	`).Error; err != nil {
		t.Fatalf("create sensor_triggers table: %v", err)
	}

	return db
}

func TestGormTriggerRepository_Upsert(t *testing.T) {
	t.Run("returns not found when device is outside organization scope", func(t *testing.T) {
		db := newTestTriggerDB(t)
		repository := NewGormTriggerRepository(db)

		orgID := uint(1)
		otherOrg := uint(2)
		device := models.Device{DeviceID: "esp32-lab", OrganizationID: &otherOrg}
		if err := db.Create(&device).Error; err != nil {
			t.Fatalf("seed device: %v", err)
		}

		ctx := tenancy.WithOrganizationID(context.Background(), orgID)
		err := repository.Upsert(ctx, "esp32-lab", "temperature", models.SensorTrigger{Max: floatPtr(30)})
		if err != ErrNotFound {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("upserts trigger when device is in organization scope", func(t *testing.T) {
		db := newTestTriggerDB(t)
		repository := NewGormTriggerRepository(db)

		orgID := uint(1)
		device := models.Device{DeviceID: "esp32-lab", OrganizationID: &orgID}
		if err := db.Create(&device).Error; err != nil {
			t.Fatalf("seed device: %v", err)
		}

		ctx := tenancy.WithOrganizationID(context.Background(), orgID)
		err := repository.Upsert(ctx, "esp32-lab", "temperature", models.SensorTrigger{Max: floatPtr(30), TargetDeviceID: "pump-node-1"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		trigger, err := repository.GetByDeviceAndSensor(ctx, "esp32-lab", "temperature")
		if err != nil {
			t.Fatalf("expected no error reading back trigger, got %v", err)
		}

		if trigger.Max == nil || *trigger.Max != 30 {
			t.Fatalf("expected max 30, got %#v", trigger.Max)
		}

		if trigger.TargetDeviceID != "pump-node-1" {
			t.Fatalf("expected target device pump-node-1, got %q", trigger.TargetDeviceID)
		}
	})
}

func TestGormTriggerRepository_DeleteByDeviceAndSensor(t *testing.T) {
	t.Run("deletes only scoped trigger", func(t *testing.T) {
		db := newTestTriggerDB(t)
		repository := NewGormTriggerRepository(db)

		orgID := uint(1)
		device := models.Device{DeviceID: "esp32-lab", OrganizationID: &orgID}
		if err := db.Create(&device).Error; err != nil {
			t.Fatalf("seed device: %v", err)
		}

		trigger := models.SensorTriggerEntity{DeviceID: "esp32-lab", SensorID: "temperature", MinValue: floatPtr(10), MaxValue: floatPtr(20), UpdatedAt: time.Now().UTC()}
		if err := db.Create(&trigger).Error; err != nil {
			t.Fatalf("seed trigger: %v", err)
		}

		ctx := tenancy.WithOrganizationID(context.Background(), orgID)
		if err := repository.DeleteByDeviceAndSensor(ctx, "esp32-lab", "temperature"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		_, err := repository.GetByDeviceAndSensor(ctx, "esp32-lab", "temperature")
		if err != ErrNotFound {
			t.Fatalf("expected ErrNotFound after delete, got %v", err)
		}
	})
}

func floatPtr(value float64) *float64 {
	return &value
}
