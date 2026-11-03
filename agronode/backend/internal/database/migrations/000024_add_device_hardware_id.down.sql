DROP INDEX IF EXISTS idx_devices_hardware_id;

ALTER TABLE devices
DROP COLUMN IF EXISTS hardware_id;
