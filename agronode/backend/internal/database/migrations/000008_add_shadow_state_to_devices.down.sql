ALTER TABLE devices
DROP COLUMN IF EXISTS desired_state,
DROP COLUMN IF EXISTS reported_state;
