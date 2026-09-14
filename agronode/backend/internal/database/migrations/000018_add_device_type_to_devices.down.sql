DROP INDEX IF EXISTS idx_devices_device_type;

ALTER TABLE devices
DROP COLUMN IF EXISTS device_type;
