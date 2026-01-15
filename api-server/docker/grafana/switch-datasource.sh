#!/bin/bash
# Script to switch between local and production TimescaleDB data sources

DS_DIR="docker/grafana/provisioning/datasources"

if [ "$1" == "production" ]; then
    echo "Switching to production TimescaleDB..."
    # Backup other configs (Grafana reads ALL .yml files)
    mv "$DS_DIR/timescaledb-local.yml" "$DS_DIR/timescaledb-local.yml.bak" 2>/dev/null || true
    mv "$DS_DIR/timescaledb-production.yml.bak" "$DS_DIR/timescaledb-production.yml.bak2" 2>/dev/null || true
    
    # Use production config as the only active file
    cat > "$DS_DIR/timescaledb.yml" << 'EOF'
apiVersion: 1
datasources:
  - name: TimescaleDB Production
    type: postgres
    access: proxy
    url: tramway.proxy.rlwy.net:43259
    database: railway
    user: railway
    secureJsonData:
      password: g39zhg3gb1xis72ac7yv6clzbag3z2dl
    jsonData:
      sslmode: require
      timescaledb: true
      postgresVersion: 1600
      maxOpenConns: 100
      maxIdleConns: 100
      connMaxLifetime: 14400
    isDefault: true
    editable: true
EOF
    echo "✓ Switched to production TimescaleDB"
    echo "Restarting Grafana..."
    docker restart k8s_cost_grafana
    echo "✓ Grafana restarted. Wait a few seconds, then check http://localhost:3000"
    
elif [ "$1" == "local" ]; then
    echo "Switching to local TimescaleDB..."
    # Backup other configs (Grafana reads ALL .yml files)
    mv "$DS_DIR/timescaledb-production.yml" "$DS_DIR/timescaledb-production.yml.bak" 2>/dev/null || true
    mv "$DS_DIR/timescaledb-local.yml.bak" "$DS_DIR/timescaledb-local.yml.bak2" 2>/dev/null || true
    
    # Use local config as the only active file
    cat > "$DS_DIR/timescaledb.yml" << 'EOF'
apiVersion: 1
datasources:
  - name: TimescaleDB Local
    type: postgres
    access: proxy
    url: timescaledb:5432
    database: timeseries
    user: ts_user
    secureJsonData:
      password: ts_pass
    jsonData:
      sslmode: disable
      timescaledb: true
      postgresVersion: 1600
      maxOpenConns: 100
      maxIdleConns: 100
      connMaxLifetime: 14400
    isDefault: true
    editable: true
EOF
    echo "✓ Switched to local TimescaleDB"
    echo "Restarting Grafana..."
    docker restart k8s_cost_grafana
    echo "✓ Grafana restarted. Wait a few seconds, then check http://localhost:3000"
    
else
    echo "Usage: $0 [production|local]"
    echo ""
    echo "Examples:"
    echo "  $0 production  # Switch to production Railway database"
    echo "  $0 local      # Switch to local Docker database"
    exit 1
fi

