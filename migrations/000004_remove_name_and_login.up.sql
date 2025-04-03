-- Remove name and login columns from accounts table
ALTER TABLE accounts
    DROP COLUMN name,
    DROP COLUMN login;