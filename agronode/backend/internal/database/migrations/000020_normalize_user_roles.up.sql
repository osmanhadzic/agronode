UPDATE users
SET role = 'organization'
WHERE role IS NULL
   OR BTRIM(role) = ''
   OR LOWER(BTRIM(role)) = 'user';

ALTER TABLE users
ALTER COLUMN role SET DEFAULT 'organization';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_users_role'
    ) THEN
        ALTER TABLE users
        ADD CONSTRAINT chk_users_role
        CHECK (role IN ('admin', 'organization'));
    END IF;
END $$;
