package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bugfreev587/k8s-cost-api-server/internal/api_types"
	"github.com/gin-gonic/gin"
)

func (s *Server) healthCheckHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		response := api_types.HealthCheckResponse{ // Note: Using HealthCheckResponse directly as it's in the same package
			OverallStatus: "unhealthy",
			PostgreSQL:    "unhealthy",
			TimescaleDB:   "unhealthy",
			Redis:         "unhealthy",
		}

		overallHealthy := true

		// Ping PostgreSQL
		pgCtx, pgCancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer pgCancel()
		if err := s.postgresDB.Health(pgCtx); err == nil {
			response.PostgreSQL = "healthy"
		} else {
			overallHealthy = false
			response.Message = fmt.Sprintf("PostgreSQL unhealthy: %v", err)
		}

		// Ping TimescaleDB
		tsCtx, tsCancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer tsCancel()
		if err := s.timescaleDB.Health(tsCtx); err == nil {
			response.TimescaleDB = "healthy"
		} else {
			overallHealthy = false
			if response.Message != "" {
				response.Message += "; "
			}
			response.Message += fmt.Sprintf("TimescaleDB unhealthy: %v", err)
		}

		// Ping Redis
		redisCtx, redisCancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer redisCancel()
		// Cast to health.RedisHealthChecker and then call Ping().Err()
		if err := s.redisClient.Ping(redisCtx).Err(); err == nil {
			response.Redis = "healthy"
		} else {
			overallHealthy = false
			if response.Message != "" {
				response.Message += "; "
			}
			response.Message += fmt.Sprintf("Redis unhealthy: %v", err)
		}

		if overallHealthy {
			response.OverallStatus = "healthy"
			c.JSON(http.StatusOK, response)
		} else {
			c.JSON(http.StatusServiceUnavailable, response)
		}
	}
}
