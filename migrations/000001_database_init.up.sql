-- Create table for Accounts
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    login TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

-- Create table for API Keys
CREATE TABLE api_keys (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    key TEXT NOT NULL UNIQUE,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

-- Create table for Events
CREATE TABLE events (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    user TEXT NOT NULL,
    api_key_id UUID REFERENCES api_keys(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    payload JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

-- Indexes for performance
CREATE INDEX idx_events_account_id ON events(account_id);
CREATE INDEX idx_events_user ON events(user);
CREATE INDEX idx_events_name ON events(name);
CREATE INDEX idx_events_created_at ON events(created_at);