DROP INDEX IF EXISTS idx_devices_zone_id;
DROP INDEX IF EXISTS idx_devices_organization_id;

ALTER TABLE devices DROP COLUMN IF EXISTS zone_id;
ALTER TABLE devices DROP COLUMN IF EXISTS organization_id;

DROP INDEX IF EXISTS idx_zones_field_id;
DROP INDEX IF EXISTS idx_zones_organization_id;
DROP TABLE IF EXISTS zones;

DROP INDEX IF EXISTS idx_fields_farm_id;
DROP INDEX IF EXISTS idx_fields_organization_id;
DROP TABLE IF EXISTS fields;

DROP INDEX IF EXISTS idx_farms_organization_id;
DROP TABLE IF EXISTS farms;

DROP INDEX IF EXISTS idx_users_organization_id;
DROP TABLE IF EXISTS users;

DROP TABLE IF EXISTS organizations;
