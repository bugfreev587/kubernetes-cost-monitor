package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
)

// PlanService handles plan limit enforcement and usage tracking
type PlanService struct {
	postgresDB  *gorm.DB
	timescaleDB *pgxpool.Pool
}

// NewPlanService creates a new PlanService instance
func NewPlanService(postgresDB *gorm.DB, timescaleDB *pgxpool.Pool) *PlanService {
	return &PlanService{
		postgresDB:  postgresDB,
		timescaleDB: timescaleDB,
	}
}

// GetPlanByName retrieves a pricing plan by its name
func (s *PlanService) GetPlanByName(ctx context.Context, planName string) (*models.PricingPlan, error) {
	var plan models.PricingPlan
	if err := s.postgresDB.WithContext(ctx).Where("name = ?", planName).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("plan '%s' not found", planName)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &plan, nil
}

// GetTenantPlanLimits retrieves the plan limits for a tenant
func (s *PlanService) GetTenantPlanLimits(ctx context.Context, tenantID uint) (*models.PricingPlan, error) {
	var tenant models.Tenant
	if err := s.postgresDB.WithContext(ctx).First(&tenant, tenantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("tenant %d not found", tenantID)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// If no plan assigned, default to Starter
	planName := tenant.PricingPlan
	if planName == "" {
		planName = "Starter"
	}

	return s.GetPlanByName(ctx, planName)
}

// GetAllPlans retrieves all available pricing plans
func (s *PlanService) GetAllPlans(ctx context.Context) ([]models.PricingPlan, error) {
	var plans []models.PricingPlan
	if err := s.postgresDB.WithContext(ctx).Order("price_cents ASC").Find(&plans).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch plans: %w", err)
	}
	return plans, nil
}

// GetDistinctClusterCount returns the number of distinct clusters for a tenant
func (s *PlanService) GetDistinctClusterCount(ctx context.Context, tenantID int64) (int, error) {
	query := `
		SELECT COUNT(DISTINCT cluster_name)
		FROM pod_metrics
		WHERE tenant_id = $1 AND pod_name != '__aggregate__'
	`
	var count int
	err := s.timescaleDB.QueryRow(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count clusters: %w", err)
	}
	return count, nil
}

// GetDistinctNodeCount returns the number of distinct nodes for a tenant
func (s *PlanService) GetDistinctNodeCount(ctx context.Context, tenantID int64) (int, error) {
	query := `
		SELECT COUNT(DISTINCT node_name)
		FROM node_metrics
		WHERE tenant_id = $1
	`
	var count int
	err := s.timescaleDB.QueryRow(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count nodes: %w", err)
	}
	return count, nil
}

// GetTenantUserCount returns the number of users for a tenant
func (s *PlanService) GetTenantUserCount(ctx context.Context, tenantID uint) (int, error) {
	var count int64
	if err := s.postgresDB.WithContext(ctx).Model(&models.User{}).Where("tenant_id = ?", tenantID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return int(count), nil
}

// IsClusterKnown checks if a cluster already exists for a tenant
func (s *PlanService) IsClusterKnown(ctx context.Context, tenantID int64, clusterName string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM pod_metrics
			WHERE tenant_id = $1 AND cluster_name = $2 AND pod_name != '__aggregate__'
			LIMIT 1
		)
	`
	var exists bool
	err := s.timescaleDB.QueryRow(ctx, query, tenantID, clusterName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check cluster existence: %w", err)
	}
	return exists, nil
}

// TenantUsage represents current usage vs limits for a tenant
type TenantUsage struct {
	TenantID      uint   `json:"tenant_id"`
	PlanName      string `json:"plan_name"`
	ClusterCount  int    `json:"cluster_count"`
	ClusterLimit  int    `json:"cluster_limit"`  // -1 = unlimited
	NodeCount     int    `json:"node_count"`
	NodeLimit     int    `json:"node_limit"`     // -1 = unlimited
	UserCount     int    `json:"user_count"`
	UserLimit     int    `json:"user_limit"`     // -1 = unlimited
	RetentionDays int    `json:"retention_days"`
}

// GetTenantUsage returns the current usage and limits for a tenant
func (s *PlanService) GetTenantUsage(ctx context.Context, tenantID uint) (*TenantUsage, error) {
	plan, err := s.GetTenantPlanLimits(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	clusterCount, err := s.GetDistinctClusterCount(ctx, int64(tenantID))
	if err != nil {
		clusterCount = 0 // Don't fail if no metrics yet
	}

	nodeCount, err := s.GetDistinctNodeCount(ctx, int64(tenantID))
	if err != nil {
		nodeCount = 0 // Don't fail if no metrics yet
	}

	userCount, err := s.GetTenantUserCount(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &TenantUsage{
		TenantID:      tenantID,
		PlanName:      plan.Name,
		ClusterCount:  clusterCount,
		ClusterLimit:  plan.ClusterLimit,
		NodeCount:     nodeCount,
		NodeLimit:     plan.NodeLimit,
		UserCount:     userCount,
		UserLimit:     plan.UserLimit,
		RetentionDays: plan.RetentionDays,
	}, nil
}

// IsWithinLimit checks if a count is within the limit (-1 = unlimited)
func IsWithinLimit(count, limit int) bool {
	if limit == -1 {
		return true // unlimited
	}
	return count < limit
}

// PlanLimitError represents an error when plan limits are exceeded
type PlanLimitError struct {
	LimitType string // "cluster", "node", "user"
	Current   int
	Limit     int
	PlanName  string
}

func (e *PlanLimitError) Error() string {
	return fmt.Sprintf("%s limit exceeded: plan '%s' allows %d, current count is %d",
		e.LimitType, e.PlanName, e.Limit, e.Current)
}

// CheckClusterLimit verifies if adding a new cluster would exceed the plan limit
func (s *PlanService) CheckClusterLimit(ctx context.Context, tenantID int64, clusterName string) error {
	// Check if cluster already exists (not a new cluster)
	exists, err := s.IsClusterKnown(ctx, tenantID, clusterName)
	if err != nil {
		return err
	}
	if exists {
		return nil // Existing cluster, no limit check needed
	}

	// Get plan limits
	plan, err := s.GetTenantPlanLimits(ctx, uint(tenantID))
	if err != nil {
		return err
	}

	// Unlimited plan
	if plan.ClusterLimit == -1 {
		return nil
	}

	// Count current clusters
	currentCount, err := s.GetDistinctClusterCount(ctx, tenantID)
	if err != nil {
		return err
	}

	// Check limit
	if currentCount >= plan.ClusterLimit {
		return &PlanLimitError{
			LimitType: "cluster",
			Current:   currentCount,
			Limit:     plan.ClusterLimit,
			PlanName:  plan.Name,
		}
	}

	return nil
}
