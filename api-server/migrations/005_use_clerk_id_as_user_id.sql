-- Migration: Use Clerk ID as user primary key
-- This changes users.id from auto-generated BIGSERIAL to Clerk's user ID (TEXT)
-- Run this on your PostgreSQL database

-- 1. Create a new users table with TEXT id
CREATE TABLE users_new (
  id TEXT PRIMARY KEY,  -- Clerk user ID (e.g., 'user_2lLjFe4cXYZ...')
  tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  email TEXT NOT NULL UNIQUE,
  name TEXT,
  role TEXT NOT NULL DEFAULT 'viewer',  -- 'admin', 'editor', 'viewer'
  created_at timestamptz NOT NULL DEFAULT now()
);

-- 2. Copy existing data (generating temporary clerk IDs for existing users)
-- In production, you'll want to map these to real Clerk IDs
INSERT INTO users_new (id, tenant_id, email, name, role, created_at)
SELECT
  'migrated_user_' || id::text,  -- Temporary ID for migrated users
  tenant_id,
  email,
  name,
  role,
  created_at
FROM users;

-- 3. Drop old table and rename new one
DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

-- 4. Recreate indexes
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

\echo 'Migration complete: users.id is now TEXT (Clerk ID)'
\d users
