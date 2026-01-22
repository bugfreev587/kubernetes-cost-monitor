-- Migration: Fix users table for Clerk authentication
-- Run this on your production PostgreSQL database

-- 1. Add name column to users if it doesn't exist
ALTER TABLE users ADD COLUMN IF NOT EXISTS name TEXT;

-- 2. Add role column if it doesn't exist
ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'viewer';

-- 3. Drop password_hash column if it exists (we use Clerk for auth now)
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;

-- 3. Remove UNIQUE constraint on tenants.name if it exists
-- (users might create tenants with similar names)
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_name_key;

-- 4. Ensure email is unique (not tenant_id + email)
-- First drop the old constraint if it exists
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_tenant_id_email_key;

-- Add unique constraint on email only (if not exists)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'users_email_key' AND conrelid = 'users'::regclass
    ) THEN
        ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);
    END IF;
END $$;

-- Verify the changes
\echo 'Migration complete. Verifying schema...'
\d users
\d tenants
