ALTER TABLE sensor_data
ADD COLUMN IF NOT EXISTS sensors JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE sensor_data
SET sensors = jsonb_strip_nulls(jsonb_build_object(
  'temperature', temperature,
  'humidity', humidity
))
WHERE sensors = '{}'::jsonb;

DROP INDEX IF EXISTS idx_sensor_data_event_unique;

CREATE UNIQUE INDEX IF NOT EXISTS idx_sensor_data_event_unique
ON sensor_data (device_id, created_at, sensors);
