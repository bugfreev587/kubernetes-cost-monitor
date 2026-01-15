package api

import (
	"context"
	"encoding/json" // New import for marshaling JSON
	"errors"        // Not used, but needed by HealthCheckResponse as string
	"net/http"
	"net/http/httptest"
	"testing"
	"time" // needed for context.WithTimeout

	"github.com/bugfreev587/k8s-cost-api-server/internal/api_types"      // New import for HealthCheckResponse
	"github.com/bugfreev587/k8s-cost-api-server/internal/app_interfaces" // New import for app_interfaces
	"github.com/bugfreev587/k8s-cost-api-server/internal/config"         // Needed for models.Recommendation
	"github.com/bugfreev587/k8s-cost-api-server/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8" // Still needed for redis.StatusCmd from ping method
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm" // Needed by mock PostgresDB functions that implement GetPostgresDB
)

// Mock implementations for dependencies
type mockPostgresDB struct {
	healthErr error
	// Mock other methods of PostgresService if needed by tests outside health checks
}

// Ensure mockPostgresDB implements app_interfaces.PostgresService
var _ app_interfaces.PostgresService = (*mockPostgresDB)(nil)

func (m *mockPostgresDB) Health(ctx context.Context) error { return m.healthErr }
func (m *mockPostgresDB) GetPostgresDB() *gorm.DB          { return nil } // gorm.DB still needed, implies need for gorm.io/gorm import
func (m *mockPostgresDB) GetRecommendations(c context.Context) ([]models.Recommendation, error) {
	return nil, nil
}
func (m *mockPostgresDB) DismissRecommendation(id int64) error { return nil }
func (m *mockPostgresDB) ApplyRecommendation(id int64) error   { return nil }

type mockTimescaleDB struct {
	healthErr error
	// Mock other methods of TimescaleService if needed by tests outside health checks
}

// Ensure mockTimescaleDB implements app_interfaces.TimescaleService
var _ app_interfaces.TimescaleService = (*mockTimescaleDB)(nil)

func (m *mockTimescaleDB) Health(ctx context.Context) error { return m.healthErr }
func (m *mockTimescaleDB) InsertPodMetric(ctx context.Context, timeStamp time.Time, tenantID int64, cluster, namespace, pod, node string, cpuMilli, memBytes, cpuRequest, memRequest, cpuLimit, memLimit int64) error {
	return nil
}
func (m *mockTimescaleDB) InsertNodeMetric(ctx context.Context, t time.Time, tenantID int64, cluster, node, instanceType string, cpuCap, memCap int64, hourlyCost float64) error {
	return nil
}
func (m *mockTimescaleDB) GetTimescalePool() interface{} {
	return nil
}

type mockRedisClient struct {
	pingErr error
	// Mock other methods of RedisService if needed by tests outside health checks
}

// Ensure mockRedisClient implements app_interfaces.RedisService
var _ app_interfaces.RedisService = (*mockRedisClient)(nil)

func (m *mockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx, "PING")
	if m.pingErr != nil {
		cmd.SetErr(m.pingErr)
	} else {
		cmd.SetVal("PONG")
	}
	return cmd
}

func TestHealthCheckHandler_AllHealthy(t *testing.T) {
	// Setup
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/v1/health", nil)
	c.Request = req

	mockPgDB := &mockPostgresDB{healthErr: nil}
	mockTsDB := &mockTimescaleDB{healthErr: nil}
	mockRdb := &mockRedisClient{pingErr: nil}

	testServer := NewServer(&config.Config{}, mockPgDB, mockTsDB, mockRdb, nil)

	// Call the handler
	testServer.healthCheckHandler()(c)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	expectedResponse := api_types.HealthCheckResponse{
		OverallStatus: "healthy",
		PostgreSQL:    "healthy",
		TimescaleDB:   "healthy",
		Redis:         "healthy",
	}
	expectedJSON, err := json.Marshal(expectedResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedJSON), w.Body.String())
}

func TestHealthCheckHandler_PostgresUnhealthy(t *testing.T) {
	// Setup
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/v1/health", nil)
	c.Request = req

	mockPgDB := &mockPostgresDB{healthErr: errors.New("pg error")}
	mockTsDB := &mockTimescaleDB{healthErr: nil}
	mockRdb := &mockRedisClient{pingErr: nil}

	testServer := NewServer(&config.Config{}, mockPgDB, mockTsDB, mockRdb, nil)

	// Call the handler
	testServer.healthCheckHandler()(c)

	// Assertions
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	expectedResponse := api_types.HealthCheckResponse{
		OverallStatus: "unhealthy",
		PostgreSQL:    "unhealthy",
		TimescaleDB:   "healthy",
		Redis:         "healthy",
		Message:       "PostgreSQL unhealthy: pg error",
	}
	expectedJSON, err := json.Marshal(expectedResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedJSON), w.Body.String())
}

func TestHealthCheckHandler_TimescaleUnhealthy(t *testing.T) {
	// Setup
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/v1/health", nil)
	c.Request = req

	mockPgDB := &mockPostgresDB{healthErr: nil}
	mockTsDB := &mockTimescaleDB{healthErr: errors.New("ts error")}
	mockRdb := &mockRedisClient{pingErr: nil}

	testServer := NewServer(&config.Config{}, mockPgDB, mockTsDB, mockRdb, nil)

	// Call the handler
	testServer.healthCheckHandler()(c)

	// Assertions
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	expectedResponse := api_types.HealthCheckResponse{
		OverallStatus: "unhealthy",
		PostgreSQL:    "healthy",
		TimescaleDB:   "unhealthy",
		Redis:         "healthy",
		Message:       "TimescaleDB unhealthy: ts error",
	}
	expectedJSON, err := json.Marshal(expectedResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedJSON), w.Body.String())
}

func TestHealthCheckHandler_RedisUnhealthy(t *testing.T) {
	// Setup
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/v1/health", nil)
	c.Request = req

	mockPgDB := &mockPostgresDB{healthErr: nil}
	mockTsDB := &mockTimescaleDB{healthErr: nil}
	mockRdb := &mockRedisClient{pingErr: errors.New("redis error")}

	testServer := NewServer(&config.Config{}, mockPgDB, mockTsDB, mockRdb, nil)

	// Call the handler
	testServer.healthCheckHandler()(c)

	// Assertions
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	expectedResponse := api_types.HealthCheckResponse{
		OverallStatus: "unhealthy",
		PostgreSQL:    "healthy",
		TimescaleDB:   "healthy",
		Redis:         "unhealthy",
		Message:       "Redis unhealthy: redis error",
	}
	expectedJSON, err := json.Marshal(expectedResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedJSON), w.Body.String())
}

func TestHealthCheckHandler_AllUnhealthy(t *testing.T) {
	// Setup
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/v1/health", nil)
	c.Request = req

	mockPgDB := &mockPostgresDB{healthErr: errors.New("pg error")}
	mockTsDB := &mockTimescaleDB{healthErr: errors.New("ts error")}
	mockRdb := &mockRedisClient{pingErr: errors.New("redis error")}

	testServer := NewServer(&config.Config{}, mockPgDB, mockTsDB, mockRdb, nil)

	// Call the handler
	testServer.healthCheckHandler()(c)

	// Assertions
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	expectedResponse := api_types.HealthCheckResponse{
		OverallStatus: "unhealthy",
		PostgreSQL:    "unhealthy",
		TimescaleDB:   "unhealthy",
		Redis:         "unhealthy",
		Message:       "PostgreSQL unhealthy: pg error; TimescaleDB unhealthy: ts error; Redis unhealthy: redis error",
	}
	expectedJSON, err := json.Marshal(expectedResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedJSON), w.Body.String())
}
