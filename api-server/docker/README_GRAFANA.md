# Grafana Visualization Setup

This directory contains Grafana configuration for visualizing Kubernetes cost metrics from TimescaleDB.

## Quick Start

### Start Grafana with existing services:

```bash
cd docker
docker-compose -f docker-compose.yml -f grafana/docker-compose.grafana.yml up -d grafana
```

### Access Grafana:

- URL: http://localhost:3000
- Username: `admin`
- Password: `admin` (change on first login)

### Data Sources

Grafana is pre-configured with **both** data sources:
- **TimescaleDB Production** (default) - Railway production database
- **TimescaleDB Local** - Local Docker database

To switch between them, use the switch script:
```bash
cd docker/grafana
./switch-datasource.sh production  # Use production (default)
./switch-datasource.sh local      # Use local database
```

Or manually select the data source in Grafana UI when creating queries.

## Directory Structure

```
docker/grafana/
├── docker-compose.grafana.yml    # Grafana service definition
├── GRAFANA_SETUP.md              # Detailed setup guide
├── example-queries.sql            # Example SQL queries for dashboards
├── provisioning/
│   ├── datasources/
│   │   └── timescaledb.yml       # Auto-configured TimescaleDB data source
│   └── dashboards/
│       └── dashboards.yml        # Dashboard provisioning config
└── dashboards/
    └── cost-overview.json        # Example dashboard
```

## Features

- **Pre-configured Data Source**: TimescaleDB connection automatically set up
- **Example Dashboards**: Pre-built dashboards for cost visualization
- **Example Queries**: Ready-to-use SQL queries for common visualizations

## Documentation

See [GRAFANA_SETUP.md](./grafana/GRAFANA_SETUP.md) for:
- Detailed setup instructions
- Query examples
- Dashboard configuration
- Production considerations
- Troubleshooting

## Example Visualizations

1. **Cost by Namespace** - Table showing resource usage per namespace
2. **Daily Cost Trends** - Time series graph of costs over time
3. **Cost by Cluster** - Bar chart comparing cluster costs
4. **Utilization vs Requests** - Line graph showing resource efficiency
5. **Right-Sizing Candidates** - Table of underutilized pods

## Integration with API Server

While Grafana connects directly to TimescaleDB, you can also:

1. **Use API endpoints** via Grafana's JSON API data source
2. **Create custom data source plugin** for your API
3. **Use Grafana API** to embed dashboards in your React frontend

See the main [GRAFANA_SETUP.md](./grafana/GRAFANA_SETUP.md) for more details.

