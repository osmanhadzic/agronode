ALTER TABLE sensor_triggers RENAME CONSTRAINT uq_sensor_triggers_device_sensor_id TO uq_sensor_triggers_device_sensor;

ALTER TABLE sensor_triggers RENAME COLUMN sensor_id TO sensor;
