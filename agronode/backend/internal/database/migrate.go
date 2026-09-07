package database

import (
	"context"
	"embed"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

//go:embed migrations/*.up.sql
var migrationFiles embed.FS

type migration struct {
	Version string
	File    string
}

var orderedMigrations = []migration{
	{Version: "000001", File: "migrations/000001_create_sensor_data.up.sql"},
	{Version: "000002", File: "migrations/000002_deduplicate_sensor_data_and_add_unique_index.up.sql"},
	{Version: "000003", File: "migrations/000003_add_flexible_sensors_jsonb.up.sql"},
	{Version: "000004", File: "migrations/000004_create_devices_table.up.sql"},
	{Version: "000005", File: "migrations/000005_add_discovered_sensors_to_devices.up.sql"},
	{Version: "000006", File: "migrations/000006_add_device_auth_fields.up.sql"},
	{Version: "000007", File: "migrations/000007_add_tags_to_devices.up.sql"},
	{Version: "000008", File: "migrations/000008_add_shadow_state_to_devices.up.sql"},
}

func RunMigrations(context context.Context, db *gorm.DB, logger *slog.Logger) error {
	if err := db.WithContext(context).Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`).Error; err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	for _, migration := range orderedMigrations {
		var appliedCount int64
		if err := db.WithContext(context).
			Raw("SELECT COUNT(1) FROM schema_migrations WHERE version = ?", migration.Version).
			Scan(&appliedCount).Error; err != nil {
			return fmt.Errorf("check migration %s: %w", migration.Version, err)
		}

		if appliedCount > 0 {
			continue
		}

		sqlBytes, readErr := migrationFiles.ReadFile(migration.File)
		if readErr != nil {
			return fmt.Errorf("read migration %s: %w", migration.File, readErr)
		}

		if err := db.WithContext(context).Exec(string(sqlBytes)).Error; err != nil {
			return fmt.Errorf("execute migration %s: %w", migration.Version, err)
		}

		if err := db.WithContext(context).
			Exec("INSERT INTO schema_migrations(version) VALUES (?)", migration.Version).Error; err != nil {
			return fmt.Errorf("record migration %s: %w", migration.Version, err)
		}

		logger.Info("migration applied", "version", migration.Version, "file", migration.File)
	}

	return nil
}
