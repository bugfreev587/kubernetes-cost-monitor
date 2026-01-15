package health

import (
	"context"

	"github.com/go-redis/redis/v8"
)

// DBHealthChecker defines the interface for database health checks.
type DBHealthChecker interface {
	Health(ctx context.Context) error
}

// RedisHealthChecker defines the interface for Redis health checks.
type RedisHealthChecker interface {
	Ping(ctx context.Context) *redis.StatusCmd
}
