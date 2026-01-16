-- Row-Level Security (RLS) Migration for Multi-Tenant Data Isolation
-- This ensures each tenant can only access their own data

-- ============================
-- Enable Row-Level Security
-- ============================

-- Enable RLS on pod_metrics
ALTER TABLE pod_metrics ENABLE ROW LEVEL SECURITY;

-- Enable RLS on node_metrics
ALTER TABLE node_metrics ENABLE ROW LEVEL SECURITY;

-- ============================
-- Create Policies for pod_metrics
-- ============================

-- Policy: Allow tenants to SELECT only their own data
CREATE POLICY tenant_isolation_select_pod_metrics ON pod_metrics
  FOR SELECT
  USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::BIGINT);

-- Policy: Allow tenants to INSERT only with their own tenant_id
CREATE POLICY tenant_isolation_insert_pod_metrics ON pod_metrics
  FOR INSERT
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', TRUE)::BIGINT);

-- Policy: Allow tenants to UPDATE only their own data
CREATE POLICY tenant_isolation_update_pod_metrics ON pod_metrics
  FOR UPDATE
  USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::BIGINT)
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', TRUE)::BIGINT);

-- Policy: Allow tenants to DELETE only their own data
CREATE POLICY tenant_isolation_delete_pod_metrics ON pod_metrics
  FOR DELETE
  USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::BIGINT);

-- ============================
-- Create Policies for node_metrics
-- ============================

-- Policy: Allow tenants to SELECT only their own data
CREATE POLICY tenant_isolation_select_node_metrics ON node_metrics
  FOR SELECT
  USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::BIGINT);

-- Policy: Allow tenants to INSERT only with their own tenant_id
CREATE POLICY tenant_isolation_insert_node_metrics ON node_metrics
  FOR INSERT
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', TRUE)::BIGINT);

-- Policy: Allow tenants to UPDATE only their own data
CREATE POLICY tenant_isolation_update_node_metrics ON node_metrics
  FOR UPDATE
  USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::BIGINT)
  WITH CHECK (tenant_id = current_setting('app.current_tenant_id', TRUE)::BIGINT);

-- Policy: Allow tenants to DELETE only their own data
CREATE POLICY tenant_isolation_delete_node_metrics ON node_metrics
  FOR DELETE
  USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::BIGINT);

-- ============================
-- Create Bypass Policy for Admin/Service Account
-- ============================

-- Create a special role for admin operations that bypass RLS
-- This is needed for system operations like data aggregation, monitoring, etc.

-- Create admin role (if not exists)
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'timescale_admin') THEN
    CREATE ROLE timescale_admin;
  END IF;
END
$$;

-- Grant necessary permissions to admin role
GRANT SELECT, INSERT, UPDATE, DELETE ON pod_metrics TO timescale_admin;
GRANT SELECT, INSERT, UPDATE, DELETE ON node_metrics TO timescale_admin;

-- Admin bypass policy for pod_metrics (when app.bypass_rls = 'true')
CREATE POLICY admin_bypass_pod_metrics ON pod_metrics
  FOR ALL
  USING (current_setting('app.bypass_rls', TRUE) = 'true');

-- Admin bypass policy for node_metrics (when app.bypass_rls = 'true')
CREATE POLICY admin_bypass_node_metrics ON node_metrics
  FOR ALL
  USING (current_setting('app.bypass_rls', TRUE) = 'true');

-- ============================
-- Create Helper Functions
-- ============================

-- Function to set tenant context for current session
CREATE OR REPLACE FUNCTION set_tenant_context(p_tenant_id BIGINT)
RETURNS VOID AS $$
BEGIN
  PERFORM set_config('app.current_tenant_id', p_tenant_id::TEXT, FALSE);
END;
$$ LANGUAGE plpgsql;

-- Function to enable admin mode (bypass RLS)
CREATE OR REPLACE FUNCTION enable_admin_mode()
RETURNS VOID AS $$
BEGIN
  PERFORM set_config('app.bypass_rls', 'true', FALSE);
END;
$$ LANGUAGE plpgsql;

-- Function to disable admin mode
CREATE OR REPLACE FUNCTION disable_admin_mode()
RETURNS VOID AS $$
BEGIN
  PERFORM set_config('app.bypass_rls', 'false', FALSE);
END;
$$ LANGUAGE plpgsql;

-- Function to clear tenant context
CREATE OR REPLACE FUNCTION clear_tenant_context()
RETURNS VOID AS $$
BEGIN
  PERFORM set_config('app.current_tenant_id', '', FALSE);
  PERFORM set_config('app.bypass_rls', '', FALSE);
END;
$$ LANGUAGE plpgsql;

-- ============================
-- Create Indexes for Performance
-- ============================

-- Composite indexes on tenant_id + time for faster queries
CREATE INDEX IF NOT EXISTS idx_pod_metrics_tenant_time
  ON pod_metrics (tenant_id, time DESC);

CREATE INDEX IF NOT EXISTS idx_node_metrics_tenant_time
  ON node_metrics (tenant_id, time DESC);

-- Additional indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_pod_metrics_tenant_cluster
  ON pod_metrics (tenant_id, cluster_name, time DESC);

CREATE INDEX IF NOT EXISTS idx_pod_metrics_tenant_namespace
  ON pod_metrics (tenant_id, namespace, time DESC);

CREATE INDEX IF NOT EXISTS idx_node_metrics_tenant_cluster
  ON node_metrics (tenant_id, cluster_name, time DESC);

-- ============================
-- Comments for Documentation
-- ============================

COMMENT ON POLICY tenant_isolation_select_pod_metrics ON pod_metrics IS
  'Restricts SELECT queries to only return rows where tenant_id matches current_setting(app.current_tenant_id)';

COMMENT ON POLICY tenant_isolation_select_node_metrics ON node_metrics IS
  'Restricts SELECT queries to only return rows where tenant_id matches current_setting(app.current_tenant_id)';

COMMENT ON FUNCTION set_tenant_context(BIGINT) IS
  'Sets the current tenant context for the session. Call this before executing queries: SELECT set_tenant_context(123);';

COMMENT ON FUNCTION enable_admin_mode() IS
  'Enables admin mode to bypass RLS policies. Use with caution for system operations only.';

-- ============================
-- Example Usage
-- ============================

/*
-- For application queries (tenant-specific):
SELECT set_tenant_context(1);
SELECT * FROM pod_metrics;  -- Only returns tenant 1 data

-- For admin/system queries (all tenants):
SELECT enable_admin_mode();
SELECT * FROM pod_metrics;  -- Returns all data
SELECT disable_admin_mode();

-- Clear context when done:
SELECT clear_tenant_context();
*/
