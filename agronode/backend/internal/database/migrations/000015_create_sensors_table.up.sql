CREATE TABLE IF NOT EXISTS sensors (
    id BIGSERIAL PRIMARY KEY,
    device_id TEXT NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    sensor_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_sensors_device_sensor UNIQUE (device_id, sensor_id)
);

CREATE INDEX IF NOT EXISTS idx_sensors_device_id ON sensors(device_id);

INSERT INTO sensors (device_id, sensor_id, created_at, updated_at)
SELECT DISTINCT d.device_id, sensor_name.sensor_id, NOW(), NOW()
FROM devices d
CROSS JOIN LATERAL jsonb_array_elements_text(COALESCE(d.discovered_sensors, '[]'::jsonb)) AS sensor_name(sensor_id)
WHERE sensor_name.sensor_id IS NOT NULL AND sensor_name.sensor_id <> ''
ON CONFLICT (device_id, sensor_id) DO NOTHING;

INSERT INTO sensors (device_id, sensor_id, created_at, updated_at)
SELECT DISTINCT st.device_id, st.sensor_id, NOW(), NOW()
FROM sensor_triggers st
ON CONFLICT (device_id, sensor_id) DO NOTHING;

ALTER TABLE sensor_triggers
ADD CONSTRAINT fk_sensor_triggers_device
FOREIGN KEY (device_id) REFERENCES devices(device_id) ON DELETE CASCADE;

ALTER TABLE sensor_triggers
ADD CONSTRAINT fk_sensor_triggers_sensor
FOREIGN KEY (device_id, sensor_id) REFERENCES sensors(device_id, sensor_id) ON DELETE CASCADE;
