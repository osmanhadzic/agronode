DO $$
BEGIN
    IF to_regclass('public.asset_groups_id_seq') IS NOT NULL
        AND to_regclass('public.farms_id_seq') IS NULL THEN
        ALTER SEQUENCE asset_groups_id_seq RENAME TO farms_id_seq;
    END IF;

    IF to_regclass('public.asset_sections_id_seq') IS NOT NULL
        AND to_regclass('public.fields_id_seq') IS NULL THEN
        ALTER SEQUENCE asset_sections_id_seq RENAME TO fields_id_seq;
    END IF;

    IF to_regclass('public.asset_units_id_seq') IS NOT NULL
        AND to_regclass('public.zones_id_seq') IS NULL THEN
        ALTER SEQUENCE asset_units_id_seq RENAME TO zones_id_seq;
    END IF;

    IF to_regclass('public.asset_groups') IS NOT NULL
        AND to_regclass('public.farms_id_seq') IS NOT NULL THEN
        ALTER TABLE asset_groups ALTER COLUMN id SET DEFAULT nextval('farms_id_seq'::regclass);
        ALTER SEQUENCE farms_id_seq OWNED BY asset_groups.id;
    END IF;

    IF to_regclass('public.asset_sections') IS NOT NULL
        AND to_regclass('public.fields_id_seq') IS NOT NULL THEN
        ALTER TABLE asset_sections ALTER COLUMN id SET DEFAULT nextval('fields_id_seq'::regclass);
        ALTER SEQUENCE fields_id_seq OWNED BY asset_sections.id;
    END IF;

    IF to_regclass('public.asset_units') IS NOT NULL
        AND to_regclass('public.zones_id_seq') IS NOT NULL THEN
        ALTER TABLE asset_units ALTER COLUMN id SET DEFAULT nextval('zones_id_seq'::regclass);
        ALTER SEQUENCE zones_id_seq OWNED BY asset_units.id;
    END IF;

    IF to_regclass('public.idx_asset_groups_organization_id') IS NOT NULL
        AND to_regclass('public.idx_farms_organization_id') IS NULL THEN
        ALTER INDEX idx_asset_groups_organization_id RENAME TO idx_farms_organization_id;
    END IF;

    IF to_regclass('public.idx_asset_sections_organization_id') IS NOT NULL
        AND to_regclass('public.idx_fields_organization_id') IS NULL THEN
        ALTER INDEX idx_asset_sections_organization_id RENAME TO idx_fields_organization_id;
    END IF;

    IF to_regclass('public.idx_asset_sections_asset_group_id') IS NOT NULL
        AND to_regclass('public.idx_fields_farm_id') IS NULL THEN
        ALTER INDEX idx_asset_sections_asset_group_id RENAME TO idx_fields_farm_id;
    END IF;

    IF to_regclass('public.idx_asset_units_organization_id') IS NOT NULL
        AND to_regclass('public.idx_zones_organization_id') IS NULL THEN
        ALTER INDEX idx_asset_units_organization_id RENAME TO idx_zones_organization_id;
    END IF;

    IF to_regclass('public.idx_asset_units_asset_section_id') IS NOT NULL
        AND to_regclass('public.idx_zones_field_id') IS NULL THEN
        ALTER INDEX idx_asset_units_asset_section_id RENAME TO idx_zones_field_id;
    END IF;

    IF to_regclass('public.idx_devices_asset_unit_id') IS NOT NULL
        AND to_regclass('public.idx_devices_zone_id') IS NULL THEN
        ALTER INDEX idx_devices_asset_unit_id RENAME TO idx_devices_zone_id;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'asset_groups_pkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'farms_pkey'
    ) THEN
        ALTER TABLE asset_groups RENAME CONSTRAINT asset_groups_pkey TO farms_pkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'asset_sections_pkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fields_pkey'
    ) THEN
        ALTER TABLE asset_sections RENAME CONSTRAINT asset_sections_pkey TO fields_pkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'asset_units_pkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'zones_pkey'
    ) THEN
        ALTER TABLE asset_units RENAME CONSTRAINT asset_units_pkey TO zones_pkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_asset_groups_org_name'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_farms_org_name'
    ) THEN
        ALTER TABLE asset_groups RENAME CONSTRAINT uq_asset_groups_org_name TO uq_farms_org_name;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_asset_sections_org_name'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_fields_org_name'
    ) THEN
        ALTER TABLE asset_sections RENAME CONSTRAINT uq_asset_sections_org_name TO uq_fields_org_name;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_asset_units_asset_section_name'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_zones_field_name'
    ) THEN
        ALTER TABLE asset_units RENAME CONSTRAINT uq_asset_units_asset_section_name TO uq_zones_field_name;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'asset_groups_organization_id_fkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'farms_organization_id_fkey'
    ) THEN
        ALTER TABLE asset_groups RENAME CONSTRAINT asset_groups_organization_id_fkey TO farms_organization_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'asset_sections_organization_id_fkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fields_organization_id_fkey'
    ) THEN
        ALTER TABLE asset_sections RENAME CONSTRAINT asset_sections_organization_id_fkey TO fields_organization_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'asset_sections_asset_group_id_fkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fields_farm_id_fkey'
    ) THEN
        ALTER TABLE asset_sections RENAME CONSTRAINT asset_sections_asset_group_id_fkey TO fields_farm_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'asset_units_organization_id_fkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'zones_organization_id_fkey'
    ) THEN
        ALTER TABLE asset_units RENAME CONSTRAINT asset_units_organization_id_fkey TO zones_organization_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'asset_units_asset_section_id_fkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'zones_field_id_fkey'
    ) THEN
        ALTER TABLE asset_units RENAME CONSTRAINT asset_units_asset_section_id_fkey TO zones_field_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'devices_asset_unit_id_fkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'devices_zone_id_fkey'
    ) THEN
        ALTER TABLE devices RENAME CONSTRAINT devices_asset_unit_id_fkey TO devices_zone_id_fkey;
    END IF;

    IF to_regclass('public.devices') IS NOT NULL
        AND EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'devices'
              AND column_name = 'asset_unit_id'
        )
        AND NOT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'devices'
              AND column_name = 'zone_id'
        ) THEN
        ALTER TABLE devices RENAME COLUMN asset_unit_id TO zone_id;
    END IF;

    IF to_regclass('public.asset_units') IS NOT NULL
        AND EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'asset_units'
              AND column_name = 'asset_section_id'
        )
        AND NOT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'asset_units'
              AND column_name = 'field_id'
        ) THEN
        ALTER TABLE asset_units RENAME COLUMN asset_section_id TO field_id;
    END IF;

    IF to_regclass('public.asset_sections') IS NOT NULL
        AND EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'asset_sections'
              AND column_name = 'asset_group_id'
        )
        AND NOT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'asset_sections'
              AND column_name = 'farm_id'
        ) THEN
        ALTER TABLE asset_sections RENAME COLUMN asset_group_id TO farm_id;
    END IF;

    IF to_regclass('public.asset_units') IS NOT NULL AND to_regclass('public.zones') IS NULL THEN
        ALTER TABLE asset_units RENAME TO zones;
    END IF;

    IF to_regclass('public.asset_sections') IS NOT NULL AND to_regclass('public.fields') IS NULL THEN
        ALTER TABLE asset_sections RENAME TO fields;
    END IF;

    IF to_regclass('public.asset_groups') IS NOT NULL AND to_regclass('public.farms') IS NULL THEN
        ALTER TABLE asset_groups RENAME TO farms;
    END IF;
END $$;
