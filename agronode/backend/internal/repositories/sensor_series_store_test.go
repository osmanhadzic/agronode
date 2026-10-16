package repositories

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestSensorSeriesTableName(t *testing.T) {
	valid := regexp.MustCompile(`^ts_[a-z0-9_]+$`)

	t.Run("keeps plain identifiers readable", func(t *testing.T) {
		if got := SensorSeriesTableName("temperature"); got != "ts_temperature" {
			t.Fatalf("expected ts_temperature, got %q", got)
		}
	})

	t.Run("distinct ids never share a table", func(t *testing.T) {
		ids := []string{"dht11-temp", "dht11_temp", "DHT11_TEMP", "temp;DROP TABLE x", "température", ""}
		seen := map[string]string{}
		for _, id := range ids {
			name := SensorSeriesTableName(id)
			if !valid.MatchString(name) || len(name) > 63 {
				t.Fatalf("unsafe table name %q for %q", name, id)
			}
			if other, dup := seen[name]; dup {
				t.Fatalf("%q and %q map to %q", id, other, name)
			}
			seen[name] = id
		}
	})

	t.Run("long ids stay within postgres limit", func(t *testing.T) {
		name := SensorSeriesTableName(strings.Repeat("a", 200))
		if len(name) > 63 {
			t.Fatalf("name too long: %d", len(name))
		}
	})
}

func TestSensorSeriesStore_CreatesTablePerSensor(t *testing.T) {
	dsn := os.Getenv("TSDB_TEST_DSN")
	if dsn == "" {
		t.Skip("TSDB_TEST_DSN not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open timescaledb: %v", err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS sensor_series_tables (sensor_id TEXT PRIMARY KEY, table_name TEXT NOT NULL UNIQUE, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`).Error; err != nil {
		t.Fatalf("registry: %v", err)
	}

	store := NewSensorSeriesStore(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	sensors := []string{"it-temperature", "it-humidity", "it-signal_strength"}

	t.Cleanup(func() {
		for _, id := range sensors {
			db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %q", SensorSeriesTableName(id)))
			db.Exec("DELETE FROM sensor_series_tables WHERE sensor_id = ?", id)
		}
	})

	for _, id := range sensors {
		for i := 0; i < 2; i++ { // second pass checks idempotency
			if err := store.Insert(ctx, id, "dev-1", 21.5, now); err != nil {
				t.Fatalf("insert %s: %v", id, err)
			}
		}

		var rows, hypertables int64
		db.Raw(fmt.Sprintf("SELECT count(*) FROM %q", SensorSeriesTableName(id))).Scan(&rows)
		db.Raw("SELECT count(*) FROM timescaledb_information.hypertables WHERE hypertable_name = ?", SensorSeriesTableName(id)).Scan(&hypertables)
		if rows != 1 || hypertables != 1 {
			t.Fatalf("%s: rows=%d hypertables=%d", id, rows, hypertables)
		}
	}
}
