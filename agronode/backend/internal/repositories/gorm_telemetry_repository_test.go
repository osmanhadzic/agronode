package repositories

import (
	"context"
	"fmt"
	"testing"
	"time"

	"agronode/backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type telemetryRow struct {
	SensorID *string
}

func newTestTelemetryDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := db.Exec(`
		PRAGMA foreign_keys = ON;
	`).Error; err != nil {
		t.Fatalf("enable foreign keys: %v", err)
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
		CREATE TABLE sensors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id TEXT NOT NULL,
			sensor_id TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE(device_id, sensor_id),
			FOREIGN KEY(device_id) REFERENCES devices(device_id) ON DELETE CASCADE
		);
	`).Error; err != nil {
		t.Fatalf("create sensors table: %v", err)
	}

	if err := db.Exec(`
		CREATE TABLE sensor_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id TEXT NOT NULL,
			sensor_id TEXT,
			temperature REAL NOT NULL,
			humidity REAL NOT NULL,
			sensors TEXT NOT NULL,
			meta TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY(device_id) REFERENCES devices(device_id) ON DELETE CASCADE,
			FOREIGN KEY(device_id, sensor_id) REFERENCES sensors(device_id, sensor_id) ON DELETE CASCADE
		);
	`).Error; err != nil {
		t.Fatalf("create sensor_data table: %v", err)
	}

	if err := db.Exec(`
		CREATE UNIQUE INDEX idx_sensor_data_event_unique ON sensor_data(device_id, created_at, sensors);
	`).Error; err != nil {
		t.Fatalf("create sensor_data unique index: %v", err)
	}

	return db
}

func TestGormTelemetryRepository_Save(t *testing.T) {
	t.Run("persists explicit sensor id", func(t *testing.T) {
		db := newTestTelemetryDB(t)
		repository := NewGormTelemetryRepository(db)

		now := time.Now().UTC().Truncate(time.Second)
		if err := db.Create(&models.Device{DeviceID: "esp32-lab"}).Error; err != nil {
			t.Fatalf("seed device: %v", err)
		}
		if err := db.Create(&models.Sensor{DeviceID: "esp32-lab", SensorID: "temperature", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			t.Fatalf("seed sensor: %v", err)
		}

		sensorID := "temperature"
		reading := models.TelemetryReading{
			DeviceID:    "esp32-lab",
			SensorID:    sensorID,
			Temperature: 24.5,
			Humidity:    60,
			Sensors: map[string]float64{
				"temperature": 24.5,
				"humidity":    60,
			},
			CreatedAt: now,
		}

		if err := repository.Save(context.Background(), reading); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		var row telemetryRow
		if err := db.Raw(`SELECT sensor_id FROM sensor_data WHERE device_id = ? LIMIT 1`, "esp32-lab").Scan(&row).Error; err != nil {
			t.Fatalf("load sensor_data row: %v", err)
		}

		if row.SensorID == nil || *row.SensorID != "temperature" {
			t.Fatalf("expected sensor_id %q, got %#v", "temperature", row.SensorID)
		}
	})

	t.Run("infers sensor id from single sensor map", func(t *testing.T) {
		db := newTestTelemetryDB(t)
		repository := NewGormTelemetryRepository(db)

		now := time.Now().UTC().Truncate(time.Second)
		if err := db.Create(&models.Device{DeviceID: "esp32-lab"}).Error; err != nil {
			t.Fatalf("seed device: %v", err)
		}
		if err := db.Create(&models.Sensor{DeviceID: "esp32-lab", SensorID: "co2", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			t.Fatalf("seed sensor: %v", err)
		}

		reading := models.TelemetryReading{
			DeviceID:    "esp32-lab",
			Temperature: 0,
			Humidity:    0,
			Sensors: map[string]float64{
				"co2": 420,
			},
			CreatedAt: now,
		}

		if err := repository.Save(context.Background(), reading); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		var row telemetryRow
		if err := db.Raw(`SELECT sensor_id FROM sensor_data WHERE device_id = ? LIMIT 1`, "esp32-lab").Scan(&row).Error; err != nil {
			t.Fatalf("load sensor_data row: %v", err)
		}

		if row.SensorID == nil || *row.SensorID != "co2" {
			t.Fatalf("expected sensor_id %q, got %#v", "co2", row.SensorID)
		}
	})
}
