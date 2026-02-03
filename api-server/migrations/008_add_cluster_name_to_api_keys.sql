-- Migration: Add cluster_name to api_keys table
-- Each API key is now associated with a specific cluster (1 key = 1 cluster)
-- This enables cluster limit enforcement based on pricing plans

ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS cluster_name VARCHAR(255) DEFAULT 'default-cluster';

-- Add comment for documentation
COMMENT ON COLUMN api_keys.cluster_name IS 'Name of the cluster this API key is for (1 key = 1 cluster)';

-- Create index for faster lookups by tenant and cluster name
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_cluster ON api_keys(tenant_id, cluster_name) WHERE revoked = false;
