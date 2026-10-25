ALTER TABLE devices
ADD COLUMN IF NOT EXISTS hardware_id VARCHAR(64);

CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_hardware_id
ON devices (hardware_id)
WHERE hardware_id IS NOT NULL;
