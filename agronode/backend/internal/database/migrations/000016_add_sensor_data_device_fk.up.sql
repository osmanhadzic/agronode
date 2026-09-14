ALTER TABLE sensor_data
ADD CONSTRAINT fk_sensor_data_device
FOREIGN KEY (device_id) REFERENCES devices(device_id) ON DELETE CASCADE NOT VALID;
