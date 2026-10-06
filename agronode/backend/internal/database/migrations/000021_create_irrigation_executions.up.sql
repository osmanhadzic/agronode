CREATE TABLE IF NOT EXISTS irrigation_executions (
    id BIGSERIAL PRIMARY KEY,
    rule_id TEXT NOT NULL,
    asset_unit_id BIGINT,
    device_id TEXT NOT NULL,
    triggered_at TIMESTAMPTZ NOT NULL,
    reason TEXT NOT NULL,
    status TEXT NOT NULL,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_irrigation_executions_triggered_at
    ON irrigation_executions (triggered_at DESC);

CREATE INDEX IF NOT EXISTS idx_irrigation_executions_asset_unit_id
    ON irrigation_executions (asset_unit_id);

CREATE INDEX IF NOT EXISTS idx_irrigation_executions_status
    ON irrigation_executions (status);
