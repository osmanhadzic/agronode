DROP INDEX IF EXISTS idx_sensor_data_event_unique;

ALTER TABLE sensor_data
DROP COLUMN IF EXISTS sensors;

CREATE UNIQUE INDEX IF NOT EXISTS idx_sensor_data_event_unique
ON sensor_data (device_id, created_at, temperature, humidity);
