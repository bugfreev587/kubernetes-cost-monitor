package db

import (
	"context"
	"time"

	// New import for api_types
	"github.com/bugfreev587/k8s-cost-api-server/internal/app_interfaces" // New import for app_interfaces
	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
	"gorm.io/gorm"
)

/*
the wrappers are a design pattern to connect a concrete implementation from a low-level package
(db) to an interface defined for a high-level package (api), ensuring loose coupling and
preventing circular dependencies.
*/

// PostgresServiceWrapper wraps PostgresDB to implement app_interfaces.PostgresService.
type PostgresServiceWrapper struct {
	*PostgresDB
}

// Ensure PostgresServiceWrapper implements app_interfaces.PostgresService
var _ app_interfaces.PostgresService = (*PostgresServiceWrapper)(nil)

// GetPostgresDB returns the GORM DB instance.
func (w *PostgresServiceWrapper) GetPostgresDB() *gorm.DB {
	return w.PostgresDB.GetPostgresDB()
}

// GetRecommendations retrieves recommendations.
func (w *PostgresServiceWrapper) GetRecommendations(c context.Context) ([]models.Recommendation, error) {
	recs, err := w.PostgresDB.GetRecommendations(c) // Call the underlying method
	if err != nil {
		return nil, err
	}
	return recs, nil
}

// DismissRecommendation dismisses a recommendation.
func (w *PostgresServiceWrapper) DismissRecommendation(id int64) error {
	return w.PostgresDB.DismissRecommendation(id)
}

// ApplyRecommendation applies a recommendation.
func (w *PostgresServiceWrapper) ApplyRecommendation(id int64) error {
	return w.PostgresDB.ApplyRecommendation(id)
}

// Health checks the health of the PostgreSQL database.
func (w *PostgresServiceWrapper) Health(ctx context.Context) error {
	return w.PostgresDB.Health(ctx)
}

// TimescaleServiceWrapper wraps TimescaleDB to implement app_interfaces.TimescaleService.
type TimescaleServiceWrapper struct {
	*TimescaleDB
}

// Ensure TimescaleServiceWrapper implements app_interfaces.TimescaleService
var _ app_interfaces.TimescaleService = (*TimescaleServiceWrapper)(nil)

// InsertPodMetric inserts a pod metric.
func (w *TimescaleServiceWrapper) InsertPodMetric(ctx context.Context, timeStamp time.Time, tenantID int64, cluster, namespace, pod, node string, cpuMilli, memBytes, cpuRequest, memRequest, cpuLimit, memLimit int64) error {
	return w.TimescaleDB.InsertPodMetric(ctx, timeStamp, tenantID, cluster, namespace, pod, node, cpuMilli, memBytes, cpuRequest, memRequest, cpuLimit, memLimit)
}

// InsertNodeMetric inserts a node metric.
func (w *TimescaleServiceWrapper) InsertNodeMetric(ctx context.Context, t time.Time, tenantID int64, cluster, node, instanceType string, cpuCap, memCap int64, hourlyCost float64) error {
	return w.TimescaleDB.InsertNodeMetric(ctx, t, tenantID, cluster, node, instanceType, cpuCap, memCap, hourlyCost)
}

// Health checks the health of the TimescaleDB database.
func (w *TimescaleServiceWrapper) Health(ctx context.Context) error {
	return w.TimescaleDB.Health(ctx)
}

// GetTimescalePool returns the TimescaleDB connection pool.
func (w *TimescaleServiceWrapper) GetTimescalePool() interface{} {
	return w.TimescaleDB.GetTimescalePool()
}
