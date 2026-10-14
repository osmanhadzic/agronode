-- Sensor telemetry now lives in the dedicated TimescaleDB instance.
-- A table that still holds rows is kept as sensor_data_legacy so it can be copied over.
DO $$
BEGIN
    IF to_regclass('public.sensor_data') IS NULL THEN
        RETURN;
    END IF;

    IF EXISTS (SELECT 1 FROM sensor_data LIMIT 1) THEN
        ALTER TABLE sensor_data RENAME TO sensor_data_legacy;
    ELSE
        DROP TABLE sensor_data;
    END IF;
END
$$;
