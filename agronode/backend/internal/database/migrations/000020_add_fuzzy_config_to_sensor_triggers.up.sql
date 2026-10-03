ALTER TABLE sensor_triggers
ADD COLUMN IF NOT EXISTS fuzzy_config JSONB NULL;

CREATE INDEX IF NOT EXISTS idx_sensor_triggers_fuzzy_config ON sensor_triggers USING gin (fuzzy_config jsonb_path_ops);
