ALTER TABLE sensor_data
ADD COLUMN IF NOT EXISTS sensor_id TEXT;

CREATE INDEX IF NOT EXISTS idx_sensor_data_sensor_id ON sensor_data(sensor_id);

ALTER TABLE sensor_data
ADD CONSTRAINT fk_sensor_data_sensor
FOREIGN KEY (device_id, sensor_id) REFERENCES sensors(device_id, sensor_id) ON DELETE CASCADE;
