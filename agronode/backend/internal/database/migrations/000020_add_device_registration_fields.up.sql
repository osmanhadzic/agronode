ALTER TABLE devices
ADD COLUMN IF NOT EXISTS registration_status TEXT NOT NULL DEFAULT 'active';

UPDATE devices
SET registration_status = CASE
    WHEN status = 'revoked' THEN 'revoked'
    WHEN status IN ('offline', 'unknown', 'inactive') THEN 'inactive'
    ELSE 'active'
END
WHERE registration_status IS NULL OR registration_status = '';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_devices_registration_status'
    ) THEN
        ALTER TABLE devices
        ADD CONSTRAINT chk_devices_registration_status
        CHECK (registration_status IN ('active', 'inactive', 'revoked'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_devices_registration_status ON devices (registration_status);

CREATE UNIQUE INDEX IF NOT EXISTS uq_devices_org_device_identity
ON devices (organization_id, device_id)
WHERE organization_id IS NOT NULL;
