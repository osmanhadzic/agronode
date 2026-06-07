CREATE TABLE IF NOT EXISTS devices (
    id BIGSERIAL PRIMARY KEY,
    device_id TEXT UNIQUE NOT NULL,
    status TEXT NOT NULL DEFAULT 'unknown',
    firmware_version TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_seen TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_devices_device_id ON devices (device_id);
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices (status);
CREATE INDEX IF NOT EXISTS idx_devices_last_seen ON devices (last_seen);
