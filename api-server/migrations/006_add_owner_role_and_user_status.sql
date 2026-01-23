-- Migration: Add 'owner' role and user status for RBAC
-- Roles: owner > admin > editor > viewer
-- Status: active, suspended

-- 1. Add status column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';

-- 2. Create index on status for faster queries
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

-- 3. Update existing admins who were the first user of their tenant to 'owner'
-- This identifies the first user per tenant by created_at
UPDATE users u
SET role = 'owner'
FROM (
  SELECT DISTINCT ON (tenant_id) id
  FROM users
  WHERE role = 'admin'
  ORDER BY tenant_id, created_at ASC
) first_users
WHERE u.id = first_users.id;

-- 4. Add a check constraint to ensure valid roles
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
  CHECK (role IN ('owner', 'admin', 'editor', 'viewer'));

-- 5. Add a check constraint to ensure valid status
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_status_check;
ALTER TABLE users ADD CONSTRAINT users_status_check
  CHECK (status IN ('active', 'suspended', 'pending'));

-- 6. Create a function to ensure each tenant has exactly one owner
CREATE OR REPLACE FUNCTION check_tenant_owner()
RETURNS TRIGGER AS $$
BEGIN
  -- Prevent removing the only owner
  IF OLD.role = 'owner' AND NEW.role != 'owner' THEN
    IF NOT EXISTS (
      SELECT 1 FROM users
      WHERE tenant_id = OLD.tenant_id
        AND role = 'owner'
        AND id != OLD.id
    ) THEN
      RAISE EXCEPTION 'Cannot remove the only owner of a tenant';
    END IF;
  END IF;

  -- Prevent deleting the only owner
  IF TG_OP = 'DELETE' AND OLD.role = 'owner' THEN
    IF NOT EXISTS (
      SELECT 1 FROM users
      WHERE tenant_id = OLD.tenant_id
        AND role = 'owner'
        AND id != OLD.id
    ) THEN
      RAISE EXCEPTION 'Cannot delete the only owner of a tenant';
    END IF;
  END IF;

  RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- 7. Create trigger to enforce owner protection
DROP TRIGGER IF EXISTS protect_tenant_owner ON users;
CREATE TRIGGER protect_tenant_owner
  BEFORE UPDATE OR DELETE ON users
  FOR EACH ROW
  EXECUTE FUNCTION check_tenant_owner();

\echo 'Migration complete: Added owner role and user status'
\d users
