ALTER TABLE users
DROP CONSTRAINT IF EXISTS chk_users_role;

ALTER TABLE users
ALTER COLUMN role SET DEFAULT 'user';

UPDATE users
SET role = 'user'
WHERE role = 'organization';
