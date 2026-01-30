-- Migration: Add grafana_org_id to tenants table
-- This column stores the actual Grafana organization ID for OAuth mapping
-- Required because tenant IDs don't match Grafana org IDs

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS grafana_org_id INTEGER DEFAULT 0;

-- Add comment for documentation
COMMENT ON COLUMN tenants.grafana_org_id IS 'Grafana organization ID for OAuth mapping (0 means not yet synced)';
