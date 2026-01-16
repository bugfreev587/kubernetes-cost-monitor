#!/usr/bin/env tsx
/**
 * Grafana Organization Manager
 *
 * This script manages Grafana organizations for multi-tenant SaaS.
 * Each tenant gets their own Grafana organization.
 *
 * Usage:
 *   npm install -g tsx
 *   tsx scripts/grafana-org-manager.ts create-org --tenant-id=1 --tenant-name="Acme Corp"
 *   tsx scripts/grafana-org-manager.ts sync-all
 *   tsx scripts/grafana-org-manager.ts add-user --org-id=2 --email=user@example.com --role=Editor
 */

import { config } from 'dotenv';

// Load environment variables
config();

const GRAFANA_URL = process.env.GRAFANA_URL || 'https://your-grafana.up.railway.app';
const GRAFANA_ADMIN_USER = process.env.GRAFANA_ADMIN_USER || 'admin';
const GRAFANA_ADMIN_PASSWORD = process.env.GRAFANA_ADMIN_PASSWORD || '';
const GRAFANA_API_TOKEN = process.env.GRAFANA_API_TOKEN || ''; // Preferred over basic auth

// PostgreSQL connection for reading tenants
const POSTGRES_DSN = process.env.POSTGRES_DSN || 'postgresql://costdb:costdb123@localhost:5431/k8s_costs';

interface GrafanaOrg {
  id: number;
  name: string;
}

interface GrafanaUser {
  id: number;
  email: string;
  name: string;
  login: string;
  orgId: number;
  role: string;
}

interface Tenant {
  id: number;
  name: string;
  pricing_plan: string;
  created_at: Date;
}

/**
 * Grafana API Client
 */
class GrafanaClient {
  private baseUrl: string;
  private authHeader: string;

  constructor(url: string, token?: string, username?: string, password?: string) {
    this.baseUrl = url.replace(/\/$/, '');

    if (token) {
      this.authHeader = `Bearer ${token}`;
    } else if (username && password) {
      const credentials = Buffer.from(`${username}:${password}`).toString('base64');
      this.authHeader = `Basic ${credentials}`;
    } else {
      throw new Error('Either GRAFANA_API_TOKEN or GRAFANA_ADMIN_USER/PASSWORD must be provided');
    }
  }

  private async request<T>(method: string, path: string, body?: any): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const headers: Record<string, string> = {
      'Authorization': this.authHeader,
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    };

    const response = await fetch(url, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`Grafana API error (${response.status}): ${errorText}`);
    }

    return response.json();
  }

  // Organization Management
  async listOrgs(): Promise<GrafanaOrg[]> {
    return this.request<GrafanaOrg[]>('GET', '/api/orgs');
  }

  async getOrgByName(name: string): Promise<GrafanaOrg | null> {
    const orgs = await this.listOrgs();
    return orgs.find(org => org.name === name) || null;
  }

  async createOrg(name: string): Promise<{ orgId: number; message: string }> {
    return this.request('POST', '/api/orgs', { name });
  }

  async deleteOrg(orgId: number): Promise<{ message: string }> {
    return this.request('DELETE', `/api/orgs/${orgId}`);
  }

  async updateOrg(orgId: number, name: string): Promise<{ message: string }> {
    return this.request('PUT', `/api/orgs/${orgId}`, { name });
  }

  // User Management
  async searchUsers(query?: string): Promise<GrafanaUser[]> {
    const params = query ? `?query=${encodeURIComponent(query)}` : '';
    return this.request<GrafanaUser[]>('GET', `/api/users${params}`);
  }

  async getUserByEmail(email: string): Promise<GrafanaUser | null> {
    const users = await this.searchUsers(email);
    return users.find(u => u.email.toLowerCase() === email.toLowerCase()) || null;
  }

  async addUserToOrg(orgId: number, userEmail: string, role: 'Viewer' | 'Editor' | 'Admin'): Promise<{ message: string }> {
    // Switch to the org context
    return this.request('POST', `/api/orgs/${orgId}/users`, {
      loginOrEmail: userEmail,
      role: role,
    });
  }

  async removeUserFromOrg(orgId: number, userId: number): Promise<{ message: string }> {
    return this.request('DELETE', `/api/orgs/${orgId}/users/${userId}`);
  }

  async updateUserRole(orgId: number, userId: number, role: 'Viewer' | 'Editor' | 'Admin'): Promise<{ message: string }> {
    return this.request('PATCH', `/api/orgs/${orgId}/users/${userId}`, { role });
  }

  // Dashboards
  async provisionDashboard(orgId: number, dashboard: any): Promise<any> {
    // Note: This requires switching org context or using org-specific endpoints
    return this.request('POST', '/api/dashboards/db', {
      dashboard: dashboard,
      overwrite: true,
      folderId: 0,
    });
  }
}

/**
 * Database helper to fetch tenants from PostgreSQL
 */
async function getTenants(): Promise<Tenant[]> {
  const { Client } = await import('pg');
  const client = new Client({ connectionString: POSTGRES_DSN });

  try {
    await client.connect();
    const result = await client.query('SELECT id, name, pricing_plan, created_at FROM tenants ORDER BY id');
    return result.rows;
  } finally {
    await client.end();
  }
}

async function getTenantById(tenantId: number): Promise<Tenant | null> {
  const { Client } = await import('pg');
  const client = new Client({ connectionString: POSTGRES_DSN });

  try {
    await client.connect();
    const result = await client.query('SELECT id, name, pricing_plan, created_at FROM tenants WHERE id = $1', [tenantId]);
    return result.rows[0] || null;
  } finally {
    await client.end();
  }
}

/**
 * Commands
 */

async function createOrgForTenant(client: GrafanaClient, tenantId: number, tenantName: string): Promise<void> {
  const orgName = `Tenant ${tenantId} - ${tenantName}`;

  // Check if org already exists
  const existing = await client.getOrgByName(orgName);
  if (existing) {
    console.log(`✓ Organization already exists: ${orgName} (ID: ${existing.id})`);
    return;
  }

  // Create org
  const result = await client.createOrg(orgName);
  console.log(`✓ Created organization: ${orgName} (ID: ${result.orgId})`);
}

async function syncAllTenants(client: GrafanaClient): Promise<void> {
  console.log('Fetching tenants from database...');
  const tenants = await getTenants();
  console.log(`Found ${tenants.length} tenants\n`);

  for (const tenant of tenants) {
    console.log(`Processing tenant ${tenant.id}: ${tenant.name}`);
    await createOrgForTenant(client, tenant.id, tenant.name);
  }

  console.log('\n✓ Sync complete');
}

async function addUserToOrg(client: GrafanaClient, orgId: number, email: string, role: 'Viewer' | 'Editor' | 'Admin'): Promise<void> {
  console.log(`Adding user ${email} to org ${orgId} with role ${role}...`);
  const result = await client.addUserToOrg(orgId, email, role);
  console.log(`✓ ${result.message}`);
}

async function listOrgs(client: GrafanaClient): Promise<void> {
  console.log('Fetching organizations...');
  const orgs = await client.listOrgs();

  console.log(`\nFound ${orgs.length} organizations:\n`);
  for (const org of orgs) {
    console.log(`  ${org.id}: ${org.name}`);
  }
}

async function deleteOrg(client: GrafanaClient, orgId: number): Promise<void> {
  console.log(`Deleting organization ${orgId}...`);
  const result = await client.deleteOrg(orgId);
  console.log(`✓ ${result.message}`);
}

/**
 * CLI Interface
 */

async function main() {
  const args = process.argv.slice(2);
  const command = args[0];

  // Parse flags
  const flags: Record<string, string> = {};
  args.forEach(arg => {
    if (arg.startsWith('--')) {
      const [key, value] = arg.substring(2).split('=');
      flags[key] = value || 'true';
    }
  });

  // Create Grafana client
  const client = new GrafanaClient(
    GRAFANA_URL,
    GRAFANA_API_TOKEN,
    GRAFANA_ADMIN_USER,
    GRAFANA_ADMIN_PASSWORD
  );

  try {
    switch (command) {
      case 'create-org':
        if (!flags['tenant-id'] || !flags['tenant-name']) {
          console.error('Usage: create-org --tenant-id=1 --tenant-name="Acme Corp"');
          process.exit(1);
        }
        await createOrgForTenant(client, parseInt(flags['tenant-id']), flags['tenant-name']);
        break;

      case 'sync-all':
        await syncAllTenants(client);
        break;

      case 'add-user':
        if (!flags['org-id'] || !flags['email'] || !flags['role']) {
          console.error('Usage: add-user --org-id=2 --email=user@example.com --role=Editor');
          process.exit(1);
        }
        await addUserToOrg(client, parseInt(flags['org-id']), flags['email'], flags['role'] as any);
        break;

      case 'list-orgs':
        await listOrgs(client);
        break;

      case 'delete-org':
        if (!flags['org-id']) {
          console.error('Usage: delete-org --org-id=2');
          process.exit(1);
        }
        await deleteOrg(client, parseInt(flags['org-id']));
        break;

      default:
        console.log(`
Grafana Organization Manager

Commands:
  create-org --tenant-id=1 --tenant-name="Acme Corp"    Create org for a tenant
  sync-all                                               Sync all tenants from DB
  add-user --org-id=2 --email=user@example.com --role=Editor
  list-orgs                                              List all organizations
  delete-org --org-id=2                                  Delete an organization

Environment Variables:
  GRAFANA_URL              Grafana URL (default: https://your-grafana.up.railway.app)
  GRAFANA_API_TOKEN        Grafana API token (preferred)
  GRAFANA_ADMIN_USER       Admin username (fallback)
  GRAFANA_ADMIN_PASSWORD   Admin password (fallback)
  POSTGRES_DSN             PostgreSQL connection string
        `);
        process.exit(1);
    }
  } catch (error) {
    console.error('Error:', error instanceof Error ? error.message : error);
    process.exit(1);
  }
}

// Run if executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main();
}

export { GrafanaClient, getTenants, createOrgForTenant, syncAllTenants };
