package services

import (
	"context"

	"github.com/go-redis/redis/v8"

	"github.com/bugfreev587/k8s-cost-api-server/internal/app_interfaces" // Import app_interfaces
)

// RedisClientWrapper wraps a *redis.Client to implement app_interfaces.RedisService.
type RedisClientWrapper struct {
	Client *redis.Client
}

// Ping implements the Ping method for RedisHealthChecker.
func (w *RedisClientWrapper) Ping(ctx context.Context) *redis.StatusCmd {
	return w.Client.Ping(ctx)
}

// Ensure RedisClientWrapper implements app_interfaces.RedisService
var _ app_interfaces.RedisService = (*RedisClientWrapper)(nil)
