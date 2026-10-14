package database

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"time"

	"agronode/backend/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

//go:embed tsmigrations/*.up.sql
var timescaleMigrationFiles embed.FS

var timescaleMigrations = []migration{
	{Version: "000001", File: "tsmigrations/000001_create_sensor_data_hypertable.up.sql"},
	{Version: "000002", File: "tsmigrations/000002_create_hourly_aggregate.up.sql"},
	{Version: "000003", File: "tsmigrations/000003_create_sensor_series_registry.up.sql"},
	{Version: "000004", File: "tsmigrations/000004_drop_sensor_data.up.sql"},
}

func ConnectTimescale(context context.Context, cfg config.Config, logger *slog.Logger) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.TSDBHost,
		cfg.TSDBPort,
		cfg.TSDBUser,
		cfg.TSDBPassword,
		cfg.TSDBName,
	)

	var connection *gorm.DB
	var err error

	for attempt := 1; attempt <= 10; attempt++ {
		connection, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			rawDB, rawErr := connection.DB()
			if rawErr == nil {
				if pingErr := rawDB.PingContext(context); pingErr == nil {
					logger.Info("timescaledb connection established", "attempt", attempt)
					return connection, nil
				} else {
					err = pingErr
				}
			}
		}

		logger.Warn("timescaledb connection retry", "attempt", attempt, "error", err)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("connect timescaledb: %w", err)
}

func RunTimescaleMigrations(context context.Context, db *gorm.DB, logger *slog.Logger) error {
	return applyMigrations(context, db, logger, timescaleMigrationFiles, timescaleMigrations)
}
