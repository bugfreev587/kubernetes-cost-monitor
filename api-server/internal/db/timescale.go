package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bugfreev587/k8s-cost-api-server/internal/app_interfaces" // Import app_interfaces
)

// Ensure TimescaleDB implements app_interfaces.TimescaleService
var _ app_interfaces.TimescaleService = (*TimescaleDB)(nil)

type TimescaleDB struct {
	pool *pgxpool.Pool
}

func InitTimescale(dsn string) (*TimescaleDB, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 20
	cfg.MinConns = 1
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}
	return &TimescaleDB{pool: pool}, nil
}

func (db *TimescaleDB) CloseDB() {
	if db.pool != nil {
		db.pool.Close()
	}
}

func (db *TimescaleDB) GetTimescalePool() interface{} {
	return db.pool
}

func (db *TimescaleDB) Health(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

func (db *TimescaleDB) InsertPodMetric(ctx context.Context, timeStamp time.Time, tenantID int64, cluster, namespace, pod, node string, cpuMilli, memBytes, cpuRequest, memRequest, cpuLimit, memLimit int64) error {
	q := `INSERT INTO pod_metrics (time, tenant_id, cluster_name, namespace, pod_name, node_name, cpu_millicores, memory_bytes, cpu_request_millicores, memory_request_bytes, cpu_limit_millicores, memory_limit_bytes) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`
	_, err := db.pool.Exec(ctx, q, timeStamp, tenantID, cluster, namespace, pod, node, cpuMilli, memBytes, cpuRequest, memRequest, cpuLimit, memLimit)
	return err
}

func (db *TimescaleDB) InsertNodeMetric(ctx context.Context, t time.Time, tenantID int64, cluster, node, instanceType string, cpuCap, memCap int64, hourlyCost float64) error {
	q := `INSERT INTO node_metrics (time, tenant_id, cluster_name, node_name, instance_type, cpu_capacity, memory_capacity, hourly_cost_usd) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	_, err := db.pool.Exec(ctx, q, t, tenantID, cluster, node, instanceType, cpuCap, memCap, hourlyCost)
	return err
}
