DROP INDEX IF EXISTS uq_devices_org_device_identity;
DROP INDEX IF EXISTS idx_devices_registration_status;

ALTER TABLE devices
DROP CONSTRAINT IF EXISTS chk_devices_registration_status;

ALTER TABLE devices
DROP COLUMN IF EXISTS registration_status;
