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

func newTestSensorDB(t *testing.T) *gorm.DB {
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
		CREATE TABLE sensors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id TEXT NOT NULL,
			sensor_id TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE(device_id, sensor_id)
		);
	`).Error; err != nil {
		t.Fatalf("create sensors table: %v", err)
	}

	return db
}

func TestGormSensorRepository_Upsert(t *testing.T) {
	db := newTestSensorDB(t)
	repository := NewGormSensorRepository(db)

	if err := repository.Upsert(context.Background(), "esp32-lab", "temperature"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := repository.Upsert(context.Background(), "esp32-lab", "temperature"); err != nil {
		t.Fatalf("expected idempotent upsert, got %v", err)
	}

	var count int64
	if err := db.Model(&models.Sensor{}).Count(&count).Error; err != nil {
		t.Fatalf("count sensors: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 sensor row, got %d", count)
	}
}

func TestGormSensorRepository_ListByDeviceID(t *testing.T) {
	db := newTestSensorDB(t)
	repository := NewGormSensorRepository(db)

	orgID := uint(1)
	otherOrg := uint(2)
	devices := []models.Device{
		{DeviceID: "esp32-lab", OrganizationID: &orgID},
		{DeviceID: "other-device", OrganizationID: &otherOrg},
	}
	for i := range devices {
		if err := db.Create(&devices[i]).Error; err != nil {
			t.Fatalf("seed device: %v", err)
		}
	}

	now := time.Now().UTC()
	seed := []models.Sensor{
		{DeviceID: "esp32-lab", SensorID: "humidity", CreatedAt: now, UpdatedAt: now},
		{DeviceID: "esp32-lab", SensorID: "temperature", CreatedAt: now, UpdatedAt: now},
		{DeviceID: "other-device", SensorID: "co2", CreatedAt: now, UpdatedAt: now},
	}
	for i := range seed {
		if err := db.Create(&seed[i]).Error; err != nil {
			t.Fatalf("seed sensor: %v", err)
		}
	}

	ctx := tenancy.WithOrganizationID(context.Background(), orgID)
	sensors, err := repository.ListByDeviceID(ctx, "esp32-lab")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(sensors) != 2 {
		t.Fatalf("expected 2 sensors, got %d", len(sensors))
	}

	if sensors[0].SensorID != "humidity" || sensors[1].SensorID != "temperature" {
		t.Fatalf("expected sensors sorted by sensor_id, got %#v", sensors)
	}

	ctx = tenancy.WithOrganizationID(context.Background(), otherOrg)
	otherSensors, err := repository.ListByDeviceID(ctx, "esp32-lab")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(otherSensors) != 0 {
		t.Fatalf("expected 0 sensors outside org scope, got %d", len(otherSensors))
	}
}
