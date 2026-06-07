ALTER TABLE devices
DROP COLUMN IF EXISTS api_key_hash,
DROP COLUMN IF EXISTS provisioning_token_hash;
