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

	return db
}

func newTestTimescaleDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s-timescale?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open timescale sqlite db: %v", err)
	}

	if err := db.Exec(`CREATE TABLE sensor_series_tables (sensor_id TEXT PRIMARY KEY, table_name TEXT NOT NULL UNIQUE, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`).Error; err != nil {
		t.Fatalf("create registry: %v", err)
	}

	return db
}

func TestGormTelemetryRepository_PerSensorTables(t *testing.T) {
	relationalDB := newTestTelemetryDB(t)
	timescaleDB := newTestTimescaleDB(t)
	repository := NewGormTelemetryRepository(relationalDB, timescaleDB)
	ctx := context.Background()

	if err := relationalDB.Create(&models.Device{DeviceID: "esp32-lab"}).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	for sensorID, value := range map[string]float64{"temperature": 24.5, "humidity": 60, "signal_strength": -70} {
		reading := models.TelemetryReading{DeviceID: "esp32-lab", SensorID: sensorID, Sensors: map[string]float64{sensorID: value}, CreatedAt: now}
		for i := 0; i < 2; i++ { // second save must be idempotent
			if err := repository.Save(ctx, reading); err != nil {
				t.Fatalf("save %s: %v", sensorID, err)
			}
		}
	}

	var tableCount int64
	timescaleDB.Table("sensor_series_tables").Count(&tableCount)
	if tableCount != 3 {
		t.Fatalf("expected 3 sensor tables, got %d", tableCount)
	}

	readings, err := repository.ListByDeviceID(ctx, "esp32-lab")
	if err != nil || len(readings) != 3 {
		t.Fatalf("expected 3 readings, got %d (err %v)", len(readings), err)
	}

	bySensor, err := repository.ListByDeviceIDAndSensorID(ctx, "esp32-lab", "humidity")
	if err != nil || len(bySensor) != 1 || bySensor[0].Humidity != 60 {
		t.Fatalf("unexpected humidity readings %#v (err %v)", bySensor, err)
	}

	latest, err := repository.GetLatestByDeviceID(ctx, "esp32-lab")
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if latest.Temperature != 24.5 || latest.Humidity != 60 || latest.Sensors["signal_strength"] != -70 {
		t.Fatalf("unexpected latest reading %#v", latest)
	}

	end := now.Add(-time.Hour)
	ranged, err := repository.ListByDeviceIDWithDateRange(ctx, "esp32-lab", DateRange{EndDate: &end})
	if err != nil || len(ranged) != 0 {
		t.Fatalf("expected empty range, got %d (err %v)", len(ranged), err)
	}

	if _, err := repository.GetLatestByDeviceID(ctx, "unknown"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
