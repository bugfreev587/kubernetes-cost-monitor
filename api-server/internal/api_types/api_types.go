package api_types

import (
	"time"

	"github.com/lib/pq"
)

// HealthCheckResponse represents the response structure for the health check endpoint.
type HealthCheckResponse struct {
	OverallStatus string `json:"overall_status"`
	PostgreSQL    string `json:"postgresql"`
	TimescaleDB   string `json:"timescaledb"`
	Redis         string `json:"redis"`
	Message       string `json:"message,omitempty"`
}

// APIKey represents an API key in the system.
type APIKey struct {
	ID         uint `gorm:"primaryKey"`
	TenantID   uint
	KeyID      string `gorm:"uniqueIndex;size:36"`
	Salt       []byte
	SecretHash []byte
	Scopes     pq.StringArray `gorm:"type:text[]"`
	Revoked    bool
	ExpiresAt  *time.Time
	CreatedAt  time.Time
}
