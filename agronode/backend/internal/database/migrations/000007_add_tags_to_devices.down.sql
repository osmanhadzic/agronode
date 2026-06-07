DROP INDEX IF EXISTS idx_devices_tags_gin;

ALTER TABLE devices
DROP COLUMN IF EXISTS tags;
