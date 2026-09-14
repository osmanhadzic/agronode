ALTER TABLE sensor_triggers RENAME COLUMN sensor TO sensor_id;

ALTER TABLE sensor_triggers RENAME CONSTRAINT uq_sensor_triggers_device_sensor TO uq_sensor_triggers_device_sensor_id;
