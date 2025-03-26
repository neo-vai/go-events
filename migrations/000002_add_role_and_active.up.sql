-- Add role and active columns to accounts table
ALTER TABLE accounts
    ADD COLUMN role TEXT NOT NULL DEFAULT 'user',
    ADD COLUMN active BOOLEAN NOT NULL DEFAULT true;

-- Add check constraint for valid roles
ALTER TABLE accounts
    ADD CONSTRAINT valid_role CHECK (role IN ('user', 'admin'));

-- Create indexes for filtering by role and active status
CREATE INDEX idx_accounts_role ON accounts(role);
CREATE INDEX idx_accounts_active ON accounts(active);