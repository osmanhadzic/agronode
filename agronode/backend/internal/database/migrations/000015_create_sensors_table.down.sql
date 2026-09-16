ALTER TABLE sensor_triggers DROP CONSTRAINT IF EXISTS fk_sensor_triggers_sensor;
ALTER TABLE sensor_triggers DROP CONSTRAINT IF EXISTS fk_sensor_triggers_device;
DROP INDEX IF EXISTS idx_sensors_device_id;
DROP TABLE IF EXISTS sensors;
