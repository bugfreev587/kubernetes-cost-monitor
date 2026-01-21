package models

import (
	"time"

	"github.com/lib/pq"
)

// PricingPlan represents a subscription tier with its limits
type PricingPlan struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Name          string         `gorm:"uniqueIndex" json:"name"`          // 'Starter', 'Premium', 'Business'
	DisplayName   string         `json:"display_name"`
	PriceCents    int            `json:"price_cents"`                      // Price in cents (0, 4900, 19900)
	ClusterLimit  int            `json:"cluster_limit"`                    // -1 = unlimited
	NodeLimit     int            `json:"node_limit"`                       // -1 = unlimited
	UserLimit     int            `json:"user_limit"`                       // -1 = unlimited
	RetentionDays int            `json:"retention_days"`
	Features      pq.StringArray `gorm:"type:text[]" json:"features"`
	CreatedAt     time.Time      `json:"created_at"`
}

type Tenant struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string
	PricingPlan string `gorm:"column:pricing_plan;default:Starter"` // 'Starter', 'Premium', 'Business'
	CreatedAt   time.Time
}

type User struct {
	ID        uint   `gorm:"primaryKey"`
	TenantID  uint
	Email     string `gorm:"uniqueIndex"`
	Name      string
	Role      string `gorm:"default:viewer"` // 'admin', 'editor', 'viewer'
	CreatedAt time.Time
}

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

type Recommendation struct {
	ID                  uint `gorm:"primaryKey"`
	TenantID            uint
	CreatedAt           time.Time
	ClusterName         string
	Namespace           string
	PodName             string
	ResourceType        string
	CurrentRequest      int64
	RecommendedRequest  int64
	PotentialSavingsUSD float64
	Confidence          float64
	Reason              string
	Status              string // open/applied/dismissed
}
