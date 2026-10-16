-- Telemetry now lives only in the per-sensor ts_* hypertables.
DROP MATERIALIZED VIEW IF EXISTS sensor_data_hourly;
DROP TABLE IF EXISTS sensor_data;
