-- Drop indexes first
DROP INDEX IF EXISTS idx_accounts_active;
DROP INDEX IF EXISTS idx_accounts_role;

-- Drop check constraint
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS valid_role;

-- Drop columns
ALTER TABLE accounts DROP COLUMN IF EXISTS active;
ALTER TABLE accounts DROP COLUMN IF EXISTS role;