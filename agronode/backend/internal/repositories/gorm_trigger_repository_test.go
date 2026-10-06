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
			firmware_version TEXT,
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
			fuzzy_config TEXT,
			updated_at DATETIME NOT NULL,
			UNIQUE(device_id, sensor_id)
		);
	`).Error; err != nil {
		t.Fatalf("create sensor_triggers table: %v", err)
	}

	if err := db.Exec(`
		CREATE TABLE irrigation_executions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			rule_id TEXT NOT NULL,
			asset_unit_id INTEGER,
			device_id TEXT NOT NULL,
			triggered_at DATETIME NOT NULL,
			reason TEXT NOT NULL,
			status TEXT NOT NULL,
			error_message TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`).Error; err != nil {
		t.Fatalf("create irrigation_executions table: %v", err)
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

func TestGormTriggerRepository_SaveExecution(t *testing.T) {
	db := newTestTriggerDB(t)
	repository := NewGormTriggerRepository(db)

	orgID := uint(1)
	assetUnitID := uint(9)
	device := models.Device{DeviceID: "pump-node-1", OrganizationID: &orgID, ZoneID: &assetUnitID}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}

	message := "broker unavailable"
	execution := models.IrrigationExecution{
		RuleID:       "above_max",
		DeviceID:     "pump-node-1",
		TriggeredAt:  time.Now().UTC(),
		Reason:       "source_device=esp32-lab sensor=co2 limit=max value=800.0000 threshold=700.0000",
		Status:       models.IrrigationExecutionStatusFailed,
		ErrorMessage: &message,
	}

	ctx := tenancy.WithOrganizationID(context.Background(), orgID)
	if err := repository.SaveExecution(ctx, execution); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var saved models.IrrigationExecution
	if err := db.Where("rule_id = ?", "above_max").First(&saved).Error; err != nil {
		t.Fatalf("read saved execution: %v", err)
	}

	if saved.AssetUnitID == nil || *saved.AssetUnitID != assetUnitID {
		t.Fatalf("expected resolved asset unit id %d, got %v", assetUnitID, saved.AssetUnitID)
	}

	if saved.ErrorMessage == nil || *saved.ErrorMessage != message {
		t.Fatalf("expected error message %q, got %v", message, saved.ErrorMessage)
	}
}

func floatPtr(value float64) *float64 {
	return &value
}
