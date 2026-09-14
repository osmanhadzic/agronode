ALTER TABLE sensor_data
DROP CONSTRAINT IF EXISTS fk_sensor_data_sensor;

DROP INDEX IF EXISTS idx_sensor_data_sensor_id;

ALTER TABLE sensor_data
DROP COLUMN IF EXISTS sensor_id;
