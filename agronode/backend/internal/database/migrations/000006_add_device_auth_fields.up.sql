ALTER TABLE devices
ADD COLUMN IF NOT EXISTS api_key_hash TEXT,
ADD COLUMN IF NOT EXISTS provisioning_token_hash TEXT;
