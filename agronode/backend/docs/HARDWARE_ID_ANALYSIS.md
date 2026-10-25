# Device `hardware_id` – Backend Analysis

Analysis only; no code was changed as part of this document.

## Stack and structure (`agronode/backend`)

- **Entry point:** `cmd/api/main.go` calls `database.ConnectPostgres`, then `database.RunMigrations`.
- **DB code (`internal/database/`):** `postgres.go` (10-attempt retry), `migrate.go`, `store.go`, `timescale.go`.
- **Models (`internal/models/`):** snake_case columns via `gorm:"column:..."` tags. `device.go` is the `Device` model. Related models: `sensor.go`, `sensor_trigger.go`, `device_event.go`, `irrigation_execution.go`, `organization.go`, `zone.go`, `field.go`, `farm.go`, `user.go`.
- **Other layers:** `internal/handlers/` (Gin, incl. `device_handler.go`, `device_stream_handler.go`) and `internal/repositories/gorm_*_repository.go`.

## Migration tooling

- **GORM `AutoMigrate()` is not used anywhere.**
- Custom runner in `internal/database/migrate.go`:
  - Embeds `migrations/*.up.sql` and applies them in the order of the hardcoded `orderedMigrations` list.
  - Applied versions are stored in `schema_migrations(version, applied_at)`.
  - Each migration runs once, without a wrapping transaction.
  - `.down.sql` files are never executed by the runner.
- File naming: `NNNNNN_description.{up,down}.sql`.
- A new migration must be added to `orderedMigrations`, otherwise it will not run.
- Pre-existing quirk: version `000020` is used twice (`add_fuzzy_config_to_sensor_triggers` and `normalize_user_roles`) and neither is in the ordered list; `000022` is the active fuzzy migration. Leave as is.

## Current `devices` schema

From migrations 000004–000008, 000012/000019, 000018:

| Column | Type |
| --- | --- |
| `id` | BIGSERIAL PK |
| `device_id` | TEXT UNIQUE NOT NULL |
| `status` | TEXT NOT NULL DEFAULT 'unknown' |
| `firmware_version` | TEXT |
| `metadata` | JSONB NOT NULL DEFAULT '{}' |
| `last_seen` | TIMESTAMPTZ |
| `created_at`, `updated_at` | TIMESTAMPTZ NOT NULL DEFAULT NOW() |
| `discovered_sensors` | JSONB NOT NULL DEFAULT '[]' |
| `api_key_hash`, `provisioning_token_hash` | TEXT |
| `tags` | JSONB NOT NULL DEFAULT '[]' (GIN index) |
| `desired_state`, `reported_state` | JSONB NOT NULL DEFAULT '{}' |
| `device_type` | TEXT NOT NULL DEFAULT 'publisher' |
| `organization_id`, `asset_unit_id` | from organization/hierarchy migrations |

Indexes: `device_id`, `status`, `last_seen`, `device_type`, `tags` (GIN).

## Current state of `hardware_id` (uncommitted)

- `internal/database/migrate.go` registers `000024`.
- `migrations/000024_add_device_hardware_id.up.sql`: `ALTER TABLE devices ADD COLUMN hardware_id VARCHAR(64);` and `CREATE UNIQUE INDEX idx_devices_hardware_id ON devices (hardware_id) WHERE hardware_id IS NOT NULL;`
- `migrations/000024_add_device_hardware_id.down.sql`: drops the index, then the column.
- `internal/models/device.go`: `HardwareID string` with `gorm:"column:hardware_id;uniqueIndex"`.

## Issues

1. **Existing records are safe:** the column is nullable with no default, so existing rows get NULL.
2. **Model/DB mismatch:** `HardwareID` is a plain `string`, so GORM inserts `''` instead of NULL. The partial unique index only ignores NULL, so a second device without a hardware ID would hit a unique violation (e.g. `Create(&models.Device{DeviceID: ...})` in repository tests).
3. **Not idempotent:** `ADD COLUMN` and `CREATE UNIQUE INDEX` lack `IF NOT EXISTS`. Other migrations use it, and the runner has no transaction, so a half-applied run would fail on retry.
4. **Redundant `uniqueIndex` GORM tag:** inert at runtime since `AutoMigrate` is unused; it can imply a non-partial index.
5. **Normalization:** ESP32 hardware IDs are usually MAC addresses. Normalize (e.g. uppercase, no separators) before lookup/insert to avoid case-only duplicates.

## Recommended approach

- Keep a single additive migration `000024`: nullable `hardware_id` plus a partial unique index, using `ADD COLUMN IF NOT EXISTS` and `CREATE UNIQUE INDEX IF NOT EXISTS`.
- Change the model field to `*string` (`json:"hardwareId,omitempty"`), and drop or document the `uniqueIndex` tag.
- Do not backfill. Existing devices stay NULL until they register or are re-provisioned.
- Do not make the column `NOT NULL`; existing devices have no value.
- Normalize hardware IDs in the registration handler.
