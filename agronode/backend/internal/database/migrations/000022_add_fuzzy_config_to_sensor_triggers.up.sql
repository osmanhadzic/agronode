ALTER TABLE sensor_triggers
ADD COLUMN IF NOT EXISTS fuzzy_config JSONB NULL;
