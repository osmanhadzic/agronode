package database

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"agronode/backend/internal/models"
	"agronode/backend/internal/repositories"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// These tests need a disposable PostgreSQL server. Set
// AGRONODE_TEST_POSTGRES_DSN to a superuser-style DSN on localhost, e.g.
//
//	postgres://postgres:testpass@127.0.0.1:55432/postgres?sslmode=disable
//
// Each test creates and drops its own database, so the DSN's database is only
// used to issue CREATE/DROP DATABASE. Without the variable the tests are skipped.
const testPostgresDSNEnv = "AGRONODE_TEST_POSTGRES_DSN"

const hardwareIDMigration = "000024"

var preHardwareIDMigrations = []migration{
	{Version: "000004", File: "migrations/000004_create_devices_table.up.sql"},
	{Version: "000005", File: "migrations/000005_add_discovered_sensors_to_devices.up.sql"},
	{Version: "000006", File: "migrations/000006_add_device_auth_fields.up.sql"},
	{Version: "000007", File: "migrations/000007_add_tags_to_devices.up.sql"},
	{Version: "000008", File: "migrations/000008_add_shadow_state_to_devices.up.sql"},
	{Version: "000018", File: "migrations/000018_add_device_type_to_devices.up.sql"},
}

var hardwareIDMigrationStep = []migration{
	{Version: hardwareIDMigration, File: "migrations/000024_add_device_hardware_id.up.sql"},
}

const hardwareIDDownFile = "migrations/000024_add_device_hardware_id.down.sql"

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newIsolatedPostgres returns a connection to a freshly created database that
// is dropped when the test ends.
func newIsolatedPostgres(t *testing.T) *gorm.DB {
	t.Helper()

	adminDSN := os.Getenv(testPostgresDSNEnv)
	if adminDSN == "" {
		t.Skipf("%s not set; skipping PostgreSQL migration tests", testPostgresDSNEnv)
	}

	parsed, err := url.Parse(adminDSN)
	if err != nil {
		t.Fatalf("parse %s: %v", testPostgresDSNEnv, err)
	}
	if host := parsed.Hostname(); host != "localhost" && host != "127.0.0.1" {
		t.Fatalf("refusing to run destructive tests against non-local host %q", host)
	}

	admin, err := gorm.Open(postgres.Open(adminDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect admin database: %v", err)
	}

	databaseName := fmt.Sprintf("agronode_migtest_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE DATABASE " + databaseName).Error; err != nil {
		t.Fatalf("create test database: %v", err)
	}

	parsed.Path = "/" + databaseName
	db, err := gorm.Open(postgres.Open(parsed.String()), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}

	t.Cleanup(func() {
		if sqlDB, closeErr := db.DB(); closeErr == nil {
			_ = sqlDB.Close()
		}
		if dropErr := admin.Exec("DROP DATABASE IF EXISTS " + databaseName).Error; dropErr != nil {
			t.Errorf("drop test database %s: %v", databaseName, dropErr)
		}
		if sqlDB, closeErr := admin.DB(); closeErr == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

// newDevicesBeforeHardwareID applies the real migrations that build the devices
// table as it exists just before 000024 and seeds two existing devices.
func newDevicesBeforeHardwareID(t *testing.T) *gorm.DB {
	t.Helper()

	db := newIsolatedPostgres(t)
	if err := applyMigrations(context.Background(), db, testLogger(), migrationFiles, preHardwareIDMigrations); err != nil {
		t.Fatalf("apply pre-000024 migrations: %v", err)
	}

	if err := db.Exec(`
		INSERT INTO devices (device_id, firmware_version) VALUES ('legacy-1', 'v1.0.0'), ('legacy-2', 'v1.1.0')
	`).Error; err != nil {
		t.Fatalf("seed existing devices: %v", err)
	}

	return db
}

func applyHardwareIDMigration(t *testing.T, db *gorm.DB) {
	t.Helper()

	if err := applyMigrations(context.Background(), db, testLogger(), migrationFiles, hardwareIDMigrationStep); err != nil {
		t.Fatalf("apply 000024: %v", err)
	}
}

func applyHardwareIDDown(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlBytes, err := os.ReadFile(hardwareIDDownFile)
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	if err := db.Exec(string(sqlBytes)).Error; err != nil {
		t.Fatalf("apply down migration: %v", err)
	}
}

func deviceColumns(t *testing.T, db *gorm.DB) map[string]string {
	t.Helper()

	var rows []struct {
		ColumnName string
		DataType   string
		MaxLength  *int
		Nullable   string
	}
	if err := db.Raw(`
		SELECT column_name, data_type, character_maximum_length AS max_length, is_nullable AS nullable
		FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'devices'
	`).Scan(&rows).Error; err != nil {
		t.Fatalf("read columns: %v", err)
	}

	columns := make(map[string]string, len(rows))
	for _, row := range rows {
		length := ""
		if row.MaxLength != nil {
			length = fmt.Sprintf("(%d)", *row.MaxLength)
		}
		columns[row.ColumnName] = row.DataType + length + "/nullable=" + row.Nullable
	}

	return columns
}

func deviceIndexes(t *testing.T, db *gorm.DB) map[string]string {
	t.Helper()

	var rows []struct {
		Indexname string
		Indexdef  string
	}
	if err := db.Raw(`
		SELECT indexname, indexdef FROM pg_indexes
		WHERE schemaname = current_schema() AND tablename = 'devices'
	`).Scan(&rows).Error; err != nil {
		t.Fatalf("read indexes: %v", err)
	}

	indexes := make(map[string]string, len(rows))
	for _, row := range rows {
		indexes[row.Indexname] = row.Indexdef
	}

	return indexes
}

func countDevices(t *testing.T, db *gorm.DB) int64 {
	t.Helper()

	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM devices").Scan(&count).Error; err != nil {
		t.Fatalf("count devices: %v", err)
	}

	return count
}

func TestMigration000024_RegisteredInOrderedMigrations(t *testing.T) {
	found := false
	for _, registered := range orderedMigrations {
		if registered.Version == hardwareIDMigration && registered.File == hardwareIDMigrationStep[0].File {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 000024 to be registered in orderedMigrations")
	}

	for _, file := range []string{hardwareIDMigrationStep[0].File} {
		if _, err := migrationFiles.ReadFile(file); err != nil {
			t.Fatalf("embedded migration %s missing: %v", file, err)
		}
	}
	if _, err := os.Stat(hardwareIDDownFile); err != nil {
		t.Fatalf("down migration missing: %v", err)
	}
}

func TestMigration000024_AddsNullableHardwareIDColumn(t *testing.T) {
	db := newDevicesBeforeHardwareID(t)
	before := deviceColumns(t, db)
	if _, exists := before["hardware_id"]; exists {
		t.Fatal("hardware_id must not exist before 000024")
	}

	applyHardwareIDMigration(t, db)

	after := deviceColumns(t, db)
	if got := after["hardware_id"]; got != "character varying(64)/nullable=YES" {
		t.Fatalf("unexpected hardware_id definition: %q", got)
	}

	for name, definition := range before {
		if after[name] != definition {
			t.Fatalf("column %s changed: %q -> %q", name, definition, after[name])
		}
	}
	if len(after) != len(before)+1 {
		t.Fatalf("expected exactly one new column, got %d -> %d", len(before), len(after))
	}

	indexDefinition := deviceIndexes(t, db)["idx_devices_hardware_id"]
	if !strings.Contains(indexDefinition, "UNIQUE") || !strings.Contains(strings.ToUpper(indexDefinition), "WHERE") {
		t.Fatalf("expected partial unique index, got %q", indexDefinition)
	}
}

func TestMigration000024_PreservesExistingDevices(t *testing.T) {
	db := newDevicesBeforeHardwareID(t)
	applyHardwareIDMigration(t, db)

	if count := countDevices(t, db); count != 2 {
		t.Fatalf("expected 2 devices preserved, got %d", count)
	}

	var rows []struct {
		DeviceID        string
		FirmwareVersion string
		HardwareID      *string
	}
	if err := db.Raw("SELECT device_id, firmware_version, hardware_id FROM devices ORDER BY device_id").Scan(&rows).Error; err != nil {
		t.Fatalf("read devices: %v", err)
	}

	for index, expected := range []struct{ id, firmware string }{{"legacy-1", "v1.0.0"}, {"legacy-2", "v1.1.0"}} {
		if rows[index].DeviceID != expected.id || rows[index].FirmwareVersion != expected.firmware || rows[index].HardwareID != nil {
			t.Fatalf("device %d not preserved with NULL hardware_id: %+v", index, rows[index])
		}
	}
}

func TestMigration000024_IsIdempotentAndRecorded(t *testing.T) {
	db := newDevicesBeforeHardwareID(t)
	applyHardwareIDMigration(t, db)

	// Simulate a run that failed after the SQL but before recording the version.
	if err := db.Exec("DELETE FROM schema_migrations WHERE version = ?", hardwareIDMigration).Error; err != nil {
		t.Fatalf("clear migration record: %v", err)
	}
	applyHardwareIDMigration(t, db)

	var recorded int64
	if err := db.Raw("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", hardwareIDMigration).Scan(&recorded).Error; err != nil {
		t.Fatalf("read schema_migrations: %v", err)
	}
	if recorded != 1 {
		t.Fatalf("expected 000024 recorded once, got %d", recorded)
	}
}

func TestMigration000024_AllowsMultipleNullHardwareIDs(t *testing.T) {
	db := newDevicesBeforeHardwareID(t)
	applyHardwareIDMigration(t, db)

	if err := db.Exec("INSERT INTO devices (device_id) VALUES ('null-a'), ('null-b'), ('null-c')").Error; err != nil {
		t.Fatalf("multiple NULL hardware_id values must be allowed: %v", err)
	}
	if count := countDevices(t, db); count != 5 {
		t.Fatalf("expected 5 devices, got %d", count)
	}
}

func TestMigration000024_RejectsDuplicateHardwareID(t *testing.T) {
	db := newDevicesBeforeHardwareID(t)
	applyHardwareIDMigration(t, db)

	if err := db.Exec("INSERT INTO devices (device_id, hardware_id) VALUES ('a', 'AABBCCDDEEFF')").Error; err != nil {
		t.Fatalf("insert first device: %v", err)
	}

	err := db.Exec("INSERT INTO devices (device_id, hardware_id) VALUES ('b', 'AABBCCDDEEFF')").Error
	if err == nil || !strings.Contains(err.Error(), "idx_devices_hardware_id") {
		t.Fatalf("expected unique violation on idx_devices_hardware_id, got %v", err)
	}

	err = db.Exec("UPDATE devices SET hardware_id = 'AABBCCDDEEFF' WHERE device_id = 'legacy-1'").Error
	if err == nil || !strings.Contains(err.Error(), "idx_devices_hardware_id") {
		t.Fatalf("expected unique violation when updating, got %v", err)
	}
}

func TestMigration000024_AllowsDifferentHardwareIDs(t *testing.T) {
	db := newDevicesBeforeHardwareID(t)
	applyHardwareIDMigration(t, db)

	if err := db.Exec(`
		INSERT INTO devices (device_id, hardware_id) VALUES ('a', 'AABBCCDDEEFF'), ('b', '112233445566')
	`).Error; err != nil {
		t.Fatalf("different hardware ids must be allowed: %v", err)
	}

	if err := db.Exec("UPDATE devices SET hardware_id = 'A1B2C3D4E5F6' WHERE device_id = 'legacy-1'").Error; err != nil {
		t.Fatalf("setting a unique hardware id on an existing device must work: %v", err)
	}
}

func TestMigration000024_RejectsOverlongHardwareID(t *testing.T) {
	db := newDevicesBeforeHardwareID(t)
	applyHardwareIDMigration(t, db)

	err := db.Exec("INSERT INTO devices (device_id, hardware_id) VALUES ('long', ?)", strings.Repeat("A", 65)).Error
	if err == nil {
		t.Fatal("expected hardware_id longer than 64 characters to be rejected")
	}
}

func TestMigration000024_DownRemovesOnlyHardwareIDColumnAndIndex(t *testing.T) {
	db := newDevicesBeforeHardwareID(t)
	columnsBefore := deviceColumns(t, db)
	indexesBefore := deviceIndexes(t, db)

	applyHardwareIDMigration(t, db)
	if err := db.Exec("INSERT INTO devices (device_id, hardware_id) VALUES ('with-id', 'AABBCCDDEEFF')").Error; err != nil {
		t.Fatalf("seed device with hardware id: %v", err)
	}

	applyHardwareIDDown(t, db)

	if columnsAfter := deviceColumns(t, db); !reflect.DeepEqual(columnsBefore, columnsAfter) {
		t.Fatalf("columns differ after down migration:\nbefore: %v\nafter:  %v", columnsBefore, columnsAfter)
	}
	if indexesAfter := deviceIndexes(t, db); !reflect.DeepEqual(indexesBefore, indexesAfter) {
		t.Fatalf("indexes differ after down migration:\nbefore: %v\nafter:  %v", indexesBefore, indexesAfter)
	}
	if count := countDevices(t, db); count != 3 {
		t.Fatalf("down migration must keep all rows, expected 3 got %d", count)
	}

	// Down must be re-runnable. Per the rollback procedure the version row is
	// removed so the runner applies 000024 again.
	applyHardwareIDDown(t, db)
	if err := db.Exec("DELETE FROM schema_migrations WHERE version = ?", hardwareIDMigration).Error; err != nil {
		t.Fatalf("clear migration record: %v", err)
	}
	applyHardwareIDMigration(t, db)
	if _, exists := deviceColumns(t, db)["hardware_id"]; !exists {
		t.Fatal("hardware_id should exist after re-applying up")
	}
}

func TestMigration000024_ModelPersistsHardwareIDInPostgres(t *testing.T) {
	db := newDevicesBeforeHardwareID(t)
	applyHardwareIDMigration(t, db)

	hardwareID := "AABBCCDDEEFF"
	withID := models.Device{DeviceID: "model-with-id", HardwareID: &hardwareID}
	withoutID := models.Device{DeviceID: "model-without-id"}
	withoutIDToo := models.Device{DeviceID: "model-without-id-2"}

	// The test table has no organization/zone columns, so persist only the fields under test.
	for _, device := range []*models.Device{&withID, &withoutID, &withoutIDToo} {
		if err := db.Model(&models.Device{}).Select("DeviceID", "HardwareID").Omit("ID").Create(device).Error; err != nil {
			t.Fatalf("create %s through the model: %v", device.DeviceID, err)
		}
	}

	var stored []struct {
		DeviceID   string
		HardwareID *string
	}
	if err := db.Raw("SELECT device_id, hardware_id FROM devices WHERE device_id LIKE 'model-%' ORDER BY device_id").Scan(&stored).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}

	if len(stored) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(stored))
	}
	if stored[0].HardwareID == nil || *stored[0].HardwareID != hardwareID {
		t.Fatalf("model did not write hardware_id column: %+v", stored[0])
	}
	if stored[1].HardwareID != nil || stored[2].HardwareID != nil {
		t.Fatalf("nil HardwareID must be stored as NULL, got %+v %+v", stored[1], stored[2])
	}
}

func TestMigration000024_ConcurrentRegistrationsOnPostgres(t *testing.T) {
	db := newDevicesBeforeHardwareID(t)
	applyHardwareIDMigration(t, db)
	repository := repositories.NewGormDeviceRepository(db)

	// Seed the columns the model writes that the pre-000024 test table lacks.
	if err := db.Exec("ALTER TABLE devices ADD COLUMN organization_id BIGINT, ADD COLUMN asset_unit_id BIGINT").Error; err != nil {
		t.Fatalf("add model columns: %v", err)
	}

	register := func(deviceID, hardwareID string) error {
		device := &models.Device{DeviceID: deviceID, HardwareID: &hardwareID, DesiredState: models.DeviceShadowState{}, ReportedState: models.DeviceShadowState{}, Tags: []string{}, DiscoveredSensors: []string{}}
		return repository.Create(context.Background(), device)
	}

	run := func(registrations []struct{ deviceID, hardwareID string }) []error {
		results := make([]error, len(registrations))
		var group sync.WaitGroup
		start := make(chan struct{})
		for index, registration := range registrations {
			group.Add(1)
			go func() {
				defer group.Done()
				<-start
				results[index] = register(registration.deviceID, registration.hardwareID)
			}()
		}
		close(start)
		group.Wait()
		return results
	}

	t.Run("same device id with different hardware ids: exactly one wins", func(t *testing.T) {
		results := run([]struct{ deviceID, hardwareID string }{
			{"race-a", "AAAAAAAAAAAA"}, {"race-a", "BBBBBBBBBBBB"}, {"race-a", "CCCCCCCCCCCC"}, {"race-a", "DDDDDDDDDDDD"},
		})

		succeeded := 0
		for _, err := range results {
			switch {
			case err == nil:
				succeeded++
			case !errors.Is(err, repositories.ErrHardwareIDMismatch):
				t.Fatalf("unexpected error: %v", err)
			}
		}
		if succeeded != 1 {
			t.Fatalf("expected exactly one successful registration, got %d (%v)", succeeded, results)
		}

		var rows int64
		if err := db.Raw("SELECT COUNT(*) FROM devices WHERE device_id = 'race-a'").Scan(&rows).Error; err != nil || rows != 1 {
			t.Fatalf("expected a single row, got %d (%v)", rows, err)
		}
	})

	t.Run("different device ids with the same hardware id: exactly one wins", func(t *testing.T) {
		results := run([]struct{ deviceID, hardwareID string }{
			{"race-b1", "EEEEEEEEEEEE"}, {"race-b2", "EEEEEEEEEEEE"}, {"race-b3", "EEEEEEEEEEEE"}, {"race-b4", "EEEEEEEEEEEE"},
		})

		succeeded := 0
		for _, err := range results {
			switch {
			case err == nil:
				succeeded++
			case !errors.Is(err, repositories.ErrDuplicateHardwareID):
				t.Fatalf("unexpected error: %v", err)
			}
		}
		if succeeded != 1 {
			t.Fatalf("expected exactly one successful registration, got %d (%v)", succeeded, results)
		}
	})
}
