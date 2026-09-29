DROP INDEX IF EXISTS idx_devices_provisioning_status;
DROP INDEX IF EXISTS uq_devices_certificate_serial;

ALTER TABLE devices
DROP COLUMN IF EXISTS certificate_expires_at,
DROP COLUMN IF EXISTS certificate_serial,
DROP COLUMN IF EXISTS provisioning_status;
