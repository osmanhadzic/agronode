ALTER TABLE devices
ADD COLUMN IF NOT EXISTS device_type TEXT NOT NULL DEFAULT 'publisher';

UPDATE devices
SET device_type = COALESCE(NULLIF(device_type, ''), 'publisher')
WHERE device_type IS NULL OR device_type = '';

CREATE INDEX IF NOT EXISTS idx_devices_device_type ON devices (device_type);
