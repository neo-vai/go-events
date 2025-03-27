-- Re-add the old key column
ALTER TABLE api_keys ADD COLUMN key TEXT;

-- Attempt to restore keys? Not possible because hash is one-way.
-- We'll leave key empty; the down migration is destructive and intended for development only.
-- In a real scenario, you'd need a backup.

-- Drop the unique constraint on key_hash
ALTER TABLE api_keys DROP CONSTRAINT api_keys_key_hash_key;

-- Drop the key_hash column
ALTER TABLE api_keys DROP COLUMN key_hash;