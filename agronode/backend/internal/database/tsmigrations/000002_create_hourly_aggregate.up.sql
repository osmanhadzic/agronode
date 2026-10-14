CREATE MATERIALIZED VIEW IF NOT EXISTS sensor_data_hourly
WITH (timescaledb.continuous) AS
SELECT time_bucket('1 hour', created_at) AS bucket,
       device_id,
       sensor_id,
       avg(temperature) AS avg_temperature,
       min(temperature) AS min_temperature,
       max(temperature) AS max_temperature,
       avg(humidity)    AS avg_humidity,
       count(*)         AS samples
FROM sensor_data
GROUP BY bucket, device_id, sensor_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy(
    'sensor_data_hourly',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => true
);
