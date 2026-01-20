package app_interfaces

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	// For Recommendation and APIKey
	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
)

// PostgresService defines the interface for PostgreSQL database operations used by the API server.
type PostgresService interface {
	Health(ctx context.Context) error
	GetPostgresDB() *gorm.DB
	GetRecommendations(c context.Context) ([]models.Recommendation, error)
	DismissRecommendation(id int64) error
	ApplyRecommendation(id int64) error
}

// TimescaleService defines the interface for TimescaleDB database operations used by the API server.
type TimescaleService interface {
	Health(ctx context.Context) error
	InsertPodMetric(ctx context.Context, timeStamp time.Time, tenantID int64, cluster, namespace, pod, node string, cpuMilli, memBytes, cpuRequest, memRequest, cpuLimit, memLimit int64) error
	InsertPodMetricWithExtras(ctx context.Context, timeStamp time.Time, tenantID int64, cluster, namespace, pod, node string, cpuMilli, memBytes, cpuRequest, memRequest, cpuLimit, memLimit int64, labels map[string]string, phase, qosClass string, containers interface{}) error
	InsertNodeMetric(ctx context.Context, t time.Time, tenantID int64, cluster, node, instanceType string, cpuCap, memCap int64, hourlyCost float64) error
	GetTimescalePool() interface{} // Returns *pgxpool.Pool but using interface{} to avoid circular dependency
}

// RedisService defines the interface for Redis operations used by the API server.
type RedisService interface {
	Ping(ctx context.Context) *redis.StatusCmd
}
