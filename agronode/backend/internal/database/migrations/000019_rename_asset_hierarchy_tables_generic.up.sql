DO $$
BEGIN
    IF to_regclass('public.farms') IS NOT NULL AND to_regclass('public.asset_groups') IS NULL THEN
        ALTER TABLE farms RENAME TO asset_groups;
    END IF;

    IF to_regclass('public.fields') IS NOT NULL AND to_regclass('public.asset_sections') IS NULL THEN
        ALTER TABLE fields RENAME TO asset_sections;
    END IF;

    IF to_regclass('public.zones') IS NOT NULL AND to_regclass('public.asset_units') IS NULL THEN
        ALTER TABLE zones RENAME TO asset_units;
    END IF;

    IF to_regclass('public.asset_sections') IS NOT NULL
        AND EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'asset_sections'
              AND column_name = 'farm_id'
        )
        AND NOT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'asset_sections'
              AND column_name = 'asset_group_id'
        ) THEN
        ALTER TABLE asset_sections RENAME COLUMN farm_id TO asset_group_id;
    END IF;

    IF to_regclass('public.asset_units') IS NOT NULL
        AND EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'asset_units'
              AND column_name = 'field_id'
        )
        AND NOT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'asset_units'
              AND column_name = 'asset_section_id'
        ) THEN
        ALTER TABLE asset_units RENAME COLUMN field_id TO asset_section_id;
    END IF;

    IF to_regclass('public.devices') IS NOT NULL
        AND EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'devices'
              AND column_name = 'zone_id'
        )
        AND NOT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'devices'
              AND column_name = 'asset_unit_id'
        ) THEN
        ALTER TABLE devices RENAME COLUMN zone_id TO asset_unit_id;
    END IF;

    IF to_regclass('public.idx_farms_organization_id') IS NOT NULL
        AND to_regclass('public.idx_asset_groups_organization_id') IS NULL THEN
        ALTER INDEX idx_farms_organization_id RENAME TO idx_asset_groups_organization_id;
    END IF;

    IF to_regclass('public.idx_fields_organization_id') IS NOT NULL
        AND to_regclass('public.idx_asset_sections_organization_id') IS NULL THEN
        ALTER INDEX idx_fields_organization_id RENAME TO idx_asset_sections_organization_id;
    END IF;

    IF to_regclass('public.idx_fields_farm_id') IS NOT NULL
        AND to_regclass('public.idx_asset_sections_asset_group_id') IS NULL THEN
        ALTER INDEX idx_fields_farm_id RENAME TO idx_asset_sections_asset_group_id;
    END IF;

    IF to_regclass('public.idx_zones_organization_id') IS NOT NULL
        AND to_regclass('public.idx_asset_units_organization_id') IS NULL THEN
        ALTER INDEX idx_zones_organization_id RENAME TO idx_asset_units_organization_id;
    END IF;

    IF to_regclass('public.idx_zones_field_id') IS NOT NULL
        AND to_regclass('public.idx_asset_units_asset_section_id') IS NULL THEN
        ALTER INDEX idx_zones_field_id RENAME TO idx_asset_units_asset_section_id;
    END IF;

    IF to_regclass('public.idx_devices_zone_id') IS NOT NULL
        AND to_regclass('public.idx_devices_asset_unit_id') IS NULL THEN
        ALTER INDEX idx_devices_zone_id RENAME TO idx_devices_asset_unit_id;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'farms_pkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'asset_groups_pkey'
    ) THEN
        ALTER TABLE asset_groups RENAME CONSTRAINT farms_pkey TO asset_groups_pkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fields_pkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'asset_sections_pkey'
    ) THEN
        ALTER TABLE asset_sections RENAME CONSTRAINT fields_pkey TO asset_sections_pkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'zones_pkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'asset_units_pkey'
    ) THEN
        ALTER TABLE asset_units RENAME CONSTRAINT zones_pkey TO asset_units_pkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_farms_org_name'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_asset_groups_org_name'
    ) THEN
        ALTER TABLE asset_groups RENAME CONSTRAINT uq_farms_org_name TO uq_asset_groups_org_name;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_fields_org_name'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_asset_sections_org_name'
    ) THEN
        ALTER TABLE asset_sections RENAME CONSTRAINT uq_fields_org_name TO uq_asset_sections_org_name;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_zones_field_name'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_asset_units_asset_section_name'
    ) THEN
        ALTER TABLE asset_units RENAME CONSTRAINT uq_zones_field_name TO uq_asset_units_asset_section_name;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'farms_organization_id_fkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'asset_groups_organization_id_fkey'
    ) THEN
        ALTER TABLE asset_groups RENAME CONSTRAINT farms_organization_id_fkey TO asset_groups_organization_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fields_organization_id_fkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'asset_sections_organization_id_fkey'
    ) THEN
        ALTER TABLE asset_sections RENAME CONSTRAINT fields_organization_id_fkey TO asset_sections_organization_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fields_farm_id_fkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'asset_sections_asset_group_id_fkey'
    ) THEN
        ALTER TABLE asset_sections RENAME CONSTRAINT fields_farm_id_fkey TO asset_sections_asset_group_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'zones_organization_id_fkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'asset_units_organization_id_fkey'
    ) THEN
        ALTER TABLE asset_units RENAME CONSTRAINT zones_organization_id_fkey TO asset_units_organization_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'zones_field_id_fkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'asset_units_asset_section_id_fkey'
    ) THEN
        ALTER TABLE asset_units RENAME CONSTRAINT zones_field_id_fkey TO asset_units_asset_section_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'devices_zone_id_fkey'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'devices_asset_unit_id_fkey'
    ) THEN
        ALTER TABLE devices RENAME CONSTRAINT devices_zone_id_fkey TO devices_asset_unit_id_fkey;
    END IF;

    IF to_regclass('public.farms_id_seq') IS NOT NULL
        AND to_regclass('public.asset_groups_id_seq') IS NULL THEN
        ALTER SEQUENCE farms_id_seq RENAME TO asset_groups_id_seq;
    END IF;

    IF to_regclass('public.fields_id_seq') IS NOT NULL
        AND to_regclass('public.asset_sections_id_seq') IS NULL THEN
        ALTER SEQUENCE fields_id_seq RENAME TO asset_sections_id_seq;
    END IF;

    IF to_regclass('public.zones_id_seq') IS NOT NULL
        AND to_regclass('public.asset_units_id_seq') IS NULL THEN
        ALTER SEQUENCE zones_id_seq RENAME TO asset_units_id_seq;
    END IF;

    IF to_regclass('public.asset_groups') IS NOT NULL AND to_regclass('public.asset_groups_id_seq') IS NOT NULL THEN
        ALTER TABLE asset_groups ALTER COLUMN id SET DEFAULT nextval('asset_groups_id_seq'::regclass);
        ALTER SEQUENCE asset_groups_id_seq OWNED BY asset_groups.id;
    END IF;

    IF to_regclass('public.asset_sections') IS NOT NULL AND to_regclass('public.asset_sections_id_seq') IS NOT NULL THEN
        ALTER TABLE asset_sections ALTER COLUMN id SET DEFAULT nextval('asset_sections_id_seq'::regclass);
        ALTER SEQUENCE asset_sections_id_seq OWNED BY asset_sections.id;
    END IF;

    IF to_regclass('public.asset_units') IS NOT NULL AND to_regclass('public.asset_units_id_seq') IS NOT NULL THEN
        ALTER TABLE asset_units ALTER COLUMN id SET DEFAULT nextval('asset_units_id_seq'::regclass);
        ALTER SEQUENCE asset_units_id_seq OWNED BY asset_units.id;
    END IF;
END $$;
