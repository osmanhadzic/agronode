-- Registry of the per-sensor hypertables (ts_*) created on demand by the backend.
CREATE TABLE IF NOT EXISTS sensor_series_tables (
    sensor_id  TEXT        PRIMARY KEY,
    table_name TEXT        NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
