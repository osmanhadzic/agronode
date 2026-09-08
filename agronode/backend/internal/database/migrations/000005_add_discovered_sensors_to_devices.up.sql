ALTER TABLE devices
ADD COLUMN IF NOT EXISTS discovered_sensors JSONB NOT NULL DEFAULT '[]'::jsonb;
