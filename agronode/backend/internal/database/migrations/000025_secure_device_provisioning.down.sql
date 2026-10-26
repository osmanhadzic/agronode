-- Manual rollback only; never executed automatically. Destroys MQTT credentials and provisioning tokens.
DROP TABLE IF EXISTS provisioning_tokens;

DROP INDEX IF EXISTS idx_devices_mqtt_username;

ALTER TABLE devices
DROP CONSTRAINT IF EXISTS chk_devices_active_has_credentials,
DROP CONSTRAINT IF EXISTS chk_devices_lifecycle_status;

ALTER TABLE devices
DROP COLUMN IF EXISTS credentials_rotated_at,
DROP COLUMN IF EXISTS registered_at,
DROP COLUMN IF EXISTS lifecycle_status,
DROP COLUMN IF EXISTS mqtt_password_hash,
DROP COLUMN IF EXISTS mqtt_username;
