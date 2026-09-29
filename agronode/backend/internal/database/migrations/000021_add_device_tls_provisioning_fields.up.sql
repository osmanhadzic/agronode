ALTER TABLE devices
ADD COLUMN IF NOT EXISTS provisioning_status TEXT NOT NULL DEFAULT 'pending',
ADD COLUMN IF NOT EXISTS certificate_serial TEXT,
ADD COLUMN IF NOT EXISTS certificate_expires_at TIMESTAMPTZ;

UPDATE devices
SET provisioning_status = 'provisioned'
WHERE api_key_hash IS NOT NULL AND api_key_hash <> '';

CREATE UNIQUE INDEX IF NOT EXISTS uq_devices_certificate_serial
ON devices (certificate_serial)
WHERE certificate_serial IS NOT NULL AND certificate_serial <> '';

CREATE INDEX IF NOT EXISTS idx_devices_provisioning_status
ON devices (provisioning_status);
