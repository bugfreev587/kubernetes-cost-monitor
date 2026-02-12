package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/bugfreev587/k8s-cost-api-server/internal/middleware"
	"github.com/bugfreev587/k8s-cost-api-server/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GET /v1/allocation
// OpenCost-compatible allocation API
//
// Query Parameters:
//   - window: Time window (required). Formats: "24h", "7d", "today", "lastweek", "2024-01-01,2024-01-07"
//   - aggregate: Grouping dimension(s). Values: "namespace", "cluster", "node", "pod", "controller", "label:<key>"
//     Multiple aggregations can be comma-separated: "namespace,label:app"
//   - step: Time bucket size for time-series results: "1h", "1d", "1w"
//   - accumulate: How to accumulate results: "true" (single result), "false", "hour", "day", "week"
//   - idle: Include idle cost allocation: "true" or "false"
//   - shareIdle: Distribute idle costs: "true", "false", "weighted"
//   - filter: Filter expressions (can be repeated). Formats: "namespace:value", "cluster:value", "label:key=value"
//   - offset: Pagination offset
//   - limit: Pagination limit (default 1000)
//
// Example requests:
//   GET /v1/allocation?window=7d&aggregate=namespace
//   GET /v1/allocation?window=24h&aggregate=namespace,label:app&idle=true&shareIdle=weighted
//   GET /v1/allocation?window=lastweek&aggregate=cluster&step=1d&accumulate=false
//   GET /v1/allocation?window=30d&aggregate=pod&filter=namespace:production&filter=cluster:prod-east
func (s *Server) getAllocation(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"status":  "error",
			"message": "no tenant context",
		})
		return
	}

	// Parse query parameters
	params := services.AllocationParams{
		Window:     c.DefaultQuery("window", "24h"),
		Aggregate:  c.DefaultQuery("aggregate", "namespace"),
		Step:       c.Query("step"),
		Accumulate: c.DefaultQuery("accumulate", "true"),
		ShareIdle:  c.Query("shareIdle"),
	}

	// Parse idle parameter
	if idleStr := c.Query("idle"); idleStr != "" {
		params.Idle = idleStr == "true" || idleStr == "1"
	}

	// Parse includeIdle (OpenCost alias)
	if includeIdleStr := c.Query("includeIdle"); includeIdleStr != "" {
		params.Idle = includeIdleStr == "true" || includeIdleStr == "1"
	}

	// Parse filters (supports multiple filter parameters)
	// OpenCost v2 style: filter=namespace:value or filter=label:key=value
	filters := c.QueryArray("filter")
	if len(filters) == 0 {
		// Also support legacy filterXxx parameters
		if ns := c.Query("filterNamespaces"); ns != "" {
			for _, n := range strings.Split(ns, ",") {
				filters = append(filters, "namespace:"+strings.TrimSpace(n))
			}
		}
		if cl := c.Query("filterClusters"); cl != "" {
			for _, c := range strings.Split(cl, ",") {
				filters = append(filters, "cluster:"+strings.TrimSpace(c))
			}
		}
		if nd := c.Query("filterNodes"); nd != "" {
			for _, n := range strings.Split(nd, ",") {
				filters = append(filters, "node:"+strings.TrimSpace(n))
			}
		}
		if lb := c.Query("filterLabels"); lb != "" {
			filters = append(filters, "label:"+lb)
		}
	}
	params.Filters = filters

	// Parse pagination
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			params.Offset = offset
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			params.Limit = limit
		}
	}

	// Get allocations
	pool := s.timescaleDB.GetTimescalePool().(*pgxpool.Pool)
	allocSvc := services.NewAllocationService(pool)
	response, err := allocSvc.GetAllocations(c.Request.Context(), int64(tenantID), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GET /v1/allocation/compute
// On-demand allocation computation (real-time, no caching)
// Same parameters as /v1/allocation but always computes fresh
func (s *Server) getAllocationCompute(c *gin.Context) {
	// Same implementation as getAllocation - in production you might add caching to getAllocation
	// and have this bypass the cache
	s.getAllocation(c)
}

// GET /v1/allocation/summary
// Returns condensed allocation summary with key metrics
func (s *Server) getAllocationSummary(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"status":  "error",
			"message": "no tenant context",
		})
		return
	}

	// Parse query parameters - simpler than full allocation
	params := services.AllocationParams{
		Window:     c.DefaultQuery("window", "24h"),
		Aggregate:  c.DefaultQuery("aggregate", "namespace"),
		Accumulate: "true", // Summary always accumulates
		Idle:       c.Query("idle") == "true",
		ShareIdle:  c.Query("shareIdle"),
	}

	// Parse filters
	params.Filters = c.QueryArray("filter")

	// Get allocations
	pool := s.timescaleDB.GetTimescalePool().(*pgxpool.Pool)
	allocSvc := services.NewAllocationService(pool)
	response, err := allocSvc.GetAllocations(c.Request.Context(), int64(tenantID), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	// Transform to summary format
	type AllocationSummaryItem struct {
		Name            string  `json:"name"`
		CPUCoreHours    float64 `json:"cpuCoreHours"`
		CPUCost         float64 `json:"cpuCost"`
		RAMByteHours    float64 `json:"ramByteHours"`
		RAMCost         float64 `json:"ramCost"`
		TotalCost       float64 `json:"totalCost"`
		TotalEfficiency float64 `json:"totalEfficiency"`
	}

	var summaryItems []AllocationSummaryItem
	var totalCost, totalCPUCost, totalRAMCost float64

	if len(response.Data) > 0 {
		for name, alloc := range response.Data[0].Allocations {
			summaryItems = append(summaryItems, AllocationSummaryItem{
				Name:            name,
				CPUCoreHours:    alloc.CPUCoreHours,
				CPUCost:         alloc.CPUCost,
				RAMByteHours:    alloc.RAMByteHours,
				RAMCost:         alloc.RAMCost,
				TotalCost:       alloc.TotalCost,
				TotalEfficiency: alloc.TotalEfficiency,
			})
			totalCost += alloc.TotalCost
			totalCPUCost += alloc.CPUCost
			totalRAMCost += alloc.RAMCost
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":   200,
		"status": "success",
		"data": gin.H{
			"items":        summaryItems,
			"totalCost":    totalCost,
			"totalCPUCost": totalCPUCost,
			"totalRAMCost": totalRAMCost,
			"window":       params.Window,
			"aggregate":    params.Aggregate,
		},
	})
}

// GET /v1/allocation/summary/topline
// Returns aggregated totals across all allocations
func (s *Server) getAllocationTopline(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"status":  "error",
			"message": "no tenant context",
		})
		return
	}

	// Parse query parameters
	params := services.AllocationParams{
		Window:     c.DefaultQuery("window", "24h"),
		Aggregate:  "cluster", // Topline aggregates at cluster level
		Accumulate: "true",
		Idle:       c.Query("idle") == "true",
		ShareIdle:  c.Query("shareIdle"),
	}

	// Get allocations
	pool := s.timescaleDB.GetTimescalePool().(*pgxpool.Pool)
	allocSvc := services.NewAllocationService(pool)
	response, err := allocSvc.GetAllocations(c.Request.Context(), int64(tenantID), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	// Aggregate totals
	var (
		totalCost           float64
		totalCPUCost        float64
		totalRAMCost        float64
		totalCPUCoreHours   float64
		totalRAMByteHours   float64
		totalIdleCost       float64
		avgEfficiency       float64
		allocationCount     int
		efficiencySum       float64
	)

	if len(response.Data) > 0 {
		totalIdleCost = response.Data[0].IdleCost
		for _, alloc := range response.Data[0].Allocations {
			if alloc.Name == "__idle__" {
				continue
			}
			totalCost += alloc.TotalCost
			totalCPUCost += alloc.CPUCost
			totalRAMCost += alloc.RAMCost
			totalCPUCoreHours += alloc.CPUCoreHours
			totalRAMByteHours += alloc.RAMByteHours
			efficiencySum += alloc.TotalEfficiency
			allocationCount++
		}
		if allocationCount > 0 {
			avgEfficiency = efficiencySum / float64(allocationCount)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":   200,
		"status": "success",
		"data": gin.H{
			"totalCost":         totalCost + totalIdleCost,
			"totalCPUCost":      totalCPUCost,
			"totalRAMCost":      totalRAMCost,
			"totalIdleCost":     totalIdleCost,
			"totalCPUCoreHours": totalCPUCoreHours,
			"totalRAMByteHours": totalRAMByteHours,
			"avgEfficiency":     avgEfficiency,
			"allocationCount":   allocationCount,
			"window":            params.Window,
		},
	})
}
