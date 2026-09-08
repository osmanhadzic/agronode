ALTER TABLE sensor_triggers
ADD COLUMN IF NOT EXISTS target_device_id TEXT NULL;
