package api

import (
	"net/http"
	"time"

	"github.com/bugfreev587/k8s-cost-api-server/internal/middleware"
	"github.com/bugfreev587/k8s-cost-api-server/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GET /v1/costs/namespaces
func (s *Server) getCostsByNamespace(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no tenant context"})
		return
	}

	// Parse query parameters
	startTimeStr := c.DefaultQuery("start_time", "")
	endTimeStr := c.DefaultQuery("end_time", "")

	var startTime, endTime time.Time
	var err error

	if startTimeStr == "" {
		startTime = time.Now().AddDate(0, 0, -7) // Default: 7 days ago
	} else {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time format, use RFC3339"})
			return
		}
	}

	if endTimeStr == "" {
		endTime = time.Now()
	} else {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time format, use RFC3339"})
			return
		}
	}

	pool := s.timescaleDB.GetTimescalePool().(*pgxpool.Pool)
	costSvc := services.NewCostService(pool)
	results, err := costSvc.CostByNamespace(c.Request.Context(), int64(tenantID), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"start_time": startTime,
		"end_time":   endTime,
		"costs":      results,
	})
}

// GET /v1/costs/clusters
func (s *Server) getCostsByCluster(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no tenant context"})
		return
	}

	// Parse query parameters
	startTimeStr := c.DefaultQuery("start_time", "")
	endTimeStr := c.DefaultQuery("end_time", "")

	var startTime, endTime time.Time
	var err error

	if startTimeStr == "" {
		startTime = time.Now().AddDate(0, 0, -7)
	} else {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time format, use RFC3339"})
			return
		}
	}

	if endTimeStr == "" {
		endTime = time.Now()
	} else {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time format, use RFC3339"})
			return
		}
	}

	pool := s.timescaleDB.GetTimescalePool().(*pgxpool.Pool)
	costSvc := services.NewCostService(pool)
	results, err := costSvc.CostByCluster(c.Request.Context(), int64(tenantID), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"start_time": startTime,
		"end_time":   endTime,
		"costs":      results,
	})
}

// GET /v1/costs/utilization
func (s *Server) getUtilizationVsRequests(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no tenant context"})
		return
	}

	// Parse query parameters
	startTimeStr := c.DefaultQuery("start_time", "")
	endTimeStr := c.DefaultQuery("end_time", "")
	namespace := c.Query("namespace")
	cluster := c.Query("cluster")

	var startTime, endTime time.Time
	var err error

	if startTimeStr == "" {
		startTime = time.Now().AddDate(0, 0, -7)
	} else {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time format, use RFC3339"})
			return
		}
	}

	if endTimeStr == "" {
		endTime = time.Now()
	} else {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time format, use RFC3339"})
			return
		}
	}

	pool := s.timescaleDB.GetTimescalePool().(*pgxpool.Pool)
	costSvc := services.NewCostService(pool)
	results, err := costSvc.UtilizationVsRequests(c.Request.Context(), int64(tenantID), startTime, endTime, namespace, cluster)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"start_time": startTime,
		"end_time":   endTime,
		"metrics":    results,
	})
}

// GET /v1/costs/trends
func (s *Server) getCostTrends(c *gin.Context) {
	tenantID, ok := middleware.GetTenantIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no tenant context"})
		return
	}

	// Parse query parameters
	startTimeStr := c.DefaultQuery("start_time", "")
	endTimeStr := c.DefaultQuery("end_time", "")
	interval := c.DefaultQuery("interval", "daily") // daily, weekly, hourly

	var startTime, endTime time.Time
	var err error

	if startTimeStr == "" {
		startTime = time.Now().AddDate(0, 0, -30) // Default: 30 days ago
	} else {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time format, use RFC3339"})
			return
		}
	}

	if endTimeStr == "" {
		endTime = time.Now()
	} else {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time format, use RFC3339"})
			return
		}
	}

	pool := s.timescaleDB.GetTimescalePool().(*pgxpool.Pool)
	costSvc := services.NewCostService(pool)
	results, err := costSvc.CostTrends(c.Request.Context(), int64(tenantID), startTime, endTime, interval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"start_time": startTime,
		"end_time":   endTime,
		"interval":   interval,
		"trends":     results,
	})
}
