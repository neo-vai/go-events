-- Add key_hash column
ALTER TABLE api_keys ADD COLUMN key_hash TEXT;

-- Populate key_hash with SHA-256 of existing keys (if any)
-- Note: This will effectively invalidate old keys because we cannot recover the original plaintext.
-- For a new project with no data, this is safe. For production, you would need a more complex migration.
UPDATE api_keys SET key_hash = encode(sha256(key::bytea), 'hex');

-- Make key_hash NOT NULL after population
ALTER TABLE api_keys ALTER COLUMN key_hash SET NOT NULL;

-- Drop the old key column
ALTER TABLE api_keys DROP COLUMN key;

-- Add unique constraint on key_hash
ALTER TABLE api_keys ADD CONSTRAINT api_keys_key_hash_key UNIQUE (key_hash);

-- Update indexes: we no longer need an index on 'key', but the unique constraint creates one automatically.