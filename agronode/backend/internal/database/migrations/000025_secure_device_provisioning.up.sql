-- Additive only: existing devices keep all data and become lifecycle_status='pending'
-- (registered, but without MQTT credentials, so they cannot authenticate to the broker).
ALTER TABLE devices
ADD COLUMN IF NOT EXISTS mqtt_username VARCHAR(64),
ADD COLUMN IF NOT EXISTS mqtt_password_hash TEXT,
ADD COLUMN IF NOT EXISTS lifecycle_status VARCHAR(16) NOT NULL DEFAULT 'pending',
ADD COLUMN IF NOT EXISTS registered_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS credentials_rotated_at TIMESTAMPTZ;

ALTER TABLE devices
ADD CONSTRAINT chk_devices_lifecycle_status
CHECK (lifecycle_status IN ('pending', 'active', 'disabled', 'revoked'));

ALTER TABLE devices
ADD CONSTRAINT chk_devices_active_has_credentials
CHECK (lifecycle_status <> 'active' OR (mqtt_username IS NOT NULL AND mqtt_password_hash IS NOT NULL));

CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_mqtt_username
ON devices (mqtt_username)
WHERE mqtt_username IS NOT NULL;

CREATE TABLE IF NOT EXISTS provisioning_tokens (
    id BIGSERIAL PRIMARY KEY,
    token_hash CHAR(64) NOT NULL,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    asset_unit_id BIGINT NULL REFERENCES asset_units(id) ON DELETE SET NULL,
    created_by VARCHAR(255) NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL,
    consumed_device_id VARCHAR(64) NULL,
    revoked_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_provisioning_tokens_token_hash UNIQUE (token_hash)
);

CREATE INDEX IF NOT EXISTS idx_provisioning_tokens_organization_id
ON provisioning_tokens (organization_id);
