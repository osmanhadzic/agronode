package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"agronode/backend/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectPostgres(context context.Context, cfg config.Config, logger *slog.Logger) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
	)

	var databaseConnection *gorm.DB
	var err error

	for attempt := 1; attempt <= 10; attempt++ {
		databaseConnection, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			rawDB, rawErr := databaseConnection.DB()
			if rawErr == nil {
				pingErr := rawDB.PingContext(context)
				if pingErr == nil {
					logger.Info("postgres connection established", "attempt", attempt)
					return databaseConnection, nil
				}

				err = pingErr
			}
		}

		logger.Warn("postgres connection retry", "attempt", attempt, "error", err)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("connect postgres: %w", err)
}
