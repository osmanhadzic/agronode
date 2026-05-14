DELETE FROM sensor_data a
USING sensor_data b
WHERE a.id < b.id
  AND a.device_id = b.device_id
  AND a.created_at = b.created_at
  AND a.temperature = b.temperature
  AND a.humidity = b.humidity;

CREATE UNIQUE INDEX IF NOT EXISTS idx_sensor_data_event_unique
ON sensor_data (device_id, created_at, temperature, humidity);
