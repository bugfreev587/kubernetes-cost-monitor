package models

import (
	"time"

	"github.com/lib/pq"
)

type Tenant struct {
	ID          uint `gorm:"primaryKey"`
	Name        string
	PricingPlan string `gorm:"column:pricing_plan"` // 'Basic', 'Standard', 'Professional', or empty
	CreatedAt   time.Time
}

type User struct {
	ID        uint `gorm:"primaryKey"`
	TenantID  uint
	Email     string `gorm:"uniqueIndex"`
	Name      string
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
