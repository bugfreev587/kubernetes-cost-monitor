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
	ID           uint   `gorm:"primaryKey"`
	Name         string
	PricingPlan  string `gorm:"column:pricing_plan;default:Starter"` // 'Starter', 'Premium', 'Business'
	GrafanaOrgID int    `gorm:"column:grafana_org_id"`               // Grafana organization ID for OAuth mapping
	CreatedAt    time.Time
}

type User struct {
	ID        string `gorm:"primaryKey;type:text"` // Clerk user ID (e.g., 'user_xxx')
	TenantID  uint
	Email     string `gorm:"uniqueIndex"`
	Name      string
	Role      string `gorm:"default:viewer"` // 'owner', 'admin', 'editor', 'viewer'
	Status    string `gorm:"default:active"` // 'active', 'suspended'
	CreatedAt time.Time
}

// Role hierarchy constants
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

// User status constants
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusPending   = "pending" // Invited but not yet signed up
)

// RoleLevel returns the numeric level of a role for comparison
// Higher level = more permissions
func RoleLevel(role string) int {
	switch role {
	case RoleOwner:
		return 4
	case RoleAdmin:
		return 3
	case RoleEditor:
		return 2
	case RoleViewer:
		return 1
	default:
		return 0
	}
}

// HasPermission checks if a user's role has at least the required role level
func (u *User) HasPermission(requiredRole string) bool {
	return RoleLevel(u.Role) >= RoleLevel(requiredRole)
}

// IsActive checks if the user is not suspended
func (u *User) IsActive() bool {
	return u.Status == StatusActive
}

type APIKey struct {
	ID          uint `gorm:"primaryKey"`
	TenantID    uint
	KeyID       string `gorm:"uniqueIndex;size:36"`
	ClusterName string `gorm:"column:cluster_name"` // Each API key is for one cluster
	Salt        []byte
	SecretHash  []byte
	Scopes      pq.StringArray `gorm:"type:text[]"`
	Revoked     bool
	ExpiresAt   *time.Time
	CreatedAt   time.Time
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
