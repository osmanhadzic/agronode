CREATE TABLE IF NOT EXISTS sensor_triggers (
    id BIGSERIAL PRIMARY KEY,
    device_id TEXT NOT NULL,
    sensor TEXT NOT NULL,
    min_value DOUBLE PRECISION NULL,
    max_value DOUBLE PRECISION NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_sensor_triggers_device_sensor UNIQUE (device_id, sensor)
);

CREATE INDEX IF NOT EXISTS idx_sensor_triggers_device_id ON sensor_triggers(device_id);
