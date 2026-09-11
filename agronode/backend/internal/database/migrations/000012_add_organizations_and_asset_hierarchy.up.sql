CREATE TABLE IF NOT EXISTS organizations (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    full_name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_users_org_email UNIQUE (organization_id, email)
);

CREATE INDEX IF NOT EXISTS idx_users_organization_id ON users(organization_id);

CREATE TABLE IF NOT EXISTS farms (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_farms_org_name UNIQUE (organization_id, name)
);

CREATE INDEX IF NOT EXISTS idx_farms_organization_id ON farms(organization_id);

CREATE TABLE IF NOT EXISTS fields (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    farm_id BIGINT NULL REFERENCES farms(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_fields_org_name UNIQUE (organization_id, name)
);

CREATE INDEX IF NOT EXISTS idx_fields_organization_id ON fields(organization_id);
CREATE INDEX IF NOT EXISTS idx_fields_farm_id ON fields(farm_id);

CREATE TABLE IF NOT EXISTS zones (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    field_id BIGINT NOT NULL REFERENCES fields(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_zones_field_name UNIQUE (field_id, name)
);

CREATE INDEX IF NOT EXISTS idx_zones_organization_id ON zones(organization_id);
CREATE INDEX IF NOT EXISTS idx_zones_field_id ON zones(field_id);

ALTER TABLE devices
ADD COLUMN IF NOT EXISTS organization_id BIGINT NULL REFERENCES organizations(id) ON DELETE SET NULL;

ALTER TABLE devices
ADD COLUMN IF NOT EXISTS zone_id BIGINT NULL REFERENCES zones(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_devices_organization_id ON devices(organization_id);
CREATE INDEX IF NOT EXISTS idx_devices_zone_id ON devices(zone_id);
