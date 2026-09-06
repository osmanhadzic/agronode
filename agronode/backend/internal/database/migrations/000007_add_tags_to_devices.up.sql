ALTER TABLE devices
ADD COLUMN IF NOT EXISTS tags JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS idx_devices_tags_gin
ON devices USING GIN (tags);
