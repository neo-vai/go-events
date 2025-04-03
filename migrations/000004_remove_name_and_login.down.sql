ALTER TABLE accounts
    ADD COLUMN name TEXT,
    ADD COLUMN login TEXT;

ALTER TABLE accounts
    ADD CONSTRAINT accounts_login_key UNIQUE (login);