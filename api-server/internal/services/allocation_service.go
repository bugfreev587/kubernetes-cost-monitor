package services

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
)

// AllocationService handles unified cost allocation queries (OpenCost-compatible)
type AllocationService struct {
	pool       *pgxpool.Pool
	postgresDB *gorm.DB
	pricingSvc *PricingService
}

// NewAllocationService creates an allocation service with default pricing
func NewAllocationService(pool *pgxpool.Pool) *AllocationService {
	return &AllocationService{pool: pool}
}

// NewAllocationServiceWithPricing creates an allocation service with dynamic pricing support
func NewAllocationServiceWithPricing(pool *pgxpool.Pool, postgresDB *gorm.DB) *AllocationService {
	return &AllocationService{
		pool:       pool,
		postgresDB: postgresDB,
		pricingSvc: NewPricingService(postgresDB),
	}
}

// getClusterPricing returns effective pricing for a cluster
func (s *AllocationService) getClusterPricing(ctx context.Context, tenantID int64, clusterName string, asOf time.Time) (cpuRate, memRate float64) {
	// Default rates
	cpuRate = DefaultCPUCostPerCoreHour
	memRate = DefaultRAMCostPerGBHour

	// If pricing service is available, get cluster-specific pricing
	if s.pricingSvc != nil {
		pricing, err := s.pricingSvc.GetEffectiveRates(ctx, uint(tenantID), clusterName, asOf)
		if err == nil && pricing != nil {
			cpuRate = pricing.CPUPerCoreHour
			memRate = pricing.MemoryPerGBHour
		}
	}

	return cpuRate, memRate
}

// getNodePricing returns pricing for a specific node (with instance-type overrides)
func (s *AllocationService) getNodePricing(ctx context.Context, tenantID int64, clusterName, nodeName string, asOf time.Time) (cpuRate, memRate float64, hasOverride bool) {
	// Start with cluster defaults
	cpuRate, memRate = s.getClusterPricing(ctx, tenantID, clusterName, asOf)
	hasOverride = false

	// If pricing service is available, check for node-specific overrides
	if s.pricingSvc != nil {
		pricing, err := s.pricingSvc.GetEffectiveRates(ctx, uint(tenantID), clusterName, asOf)
		if err == nil && pricing != nil && pricing.InstancePricing != nil {
			// Check for node-specific pricing
			if nodePricing, ok := pricing.InstancePricing[nodeName]; ok {
				if nodePricing.CPUPerCoreHour > 0 {
					cpuRate = nodePricing.CPUPerCoreHour
					hasOverride = true
				}
				if nodePricing.MemoryPerGBHour > 0 {
					memRate = nodePricing.MemoryPerGBHour
					hasOverride = true
				}
			}
		}
	}

	return cpuRate, memRate, hasOverride
}

// AllocationParams represents query parameters for the allocation API
type AllocationParams struct {
	Window     string   // "24h", "7d", "lastweek", "2024-01-01,2024-01-07"
	Aggregate  string   // "namespace", "cluster", "label:team", "node", "pod", or comma-separated
	Step       string   // "1h", "1d", "1w" - time bucket size for time-series results
	Accumulate string   // "true", "false", "hour", "day", "week" - how to accumulate results
	Idle       bool     // Include idle cost allocation
	ShareIdle  string   // "true", "false", "weighted" - how to distribute idle costs
	Filters    []string // Filter expressions: "namespace:kube-system", "cluster:prod", "label:app=nginx"
	Offset     int      // Pagination offset
	Limit      int      // Pagination limit (default 1000)
}

// Allocation represents a single allocation entry (OpenCost-compatible structure)
type Allocation struct {
	Name       string           `json:"name"`
	Properties AllocationProps  `json:"properties"`
	Window     TimeWindow       `json:"window"`
	Start      time.Time        `json:"start"`
	End        time.Time        `json:"end"`
	Minutes    float64          `json:"minutes"`

	// CPU metrics
	CPUCores            float64 `json:"cpuCores"`
	CPUCoreRequestAvg   float64 `json:"cpuCoreRequestAverage"`
	CPUCoreUsageAvg     float64 `json:"cpuCoreUsageAverage"`
	CPUCoreHours        float64 `json:"cpuCoreHours"`
	CPUCost             float64 `json:"cpuCost"`
	CPUEfficiency       float64 `json:"cpuEfficiency"`

	// RAM metrics
	RAMBytes            float64 `json:"ramBytes"`
	RAMByteRequestAvg   float64 `json:"ramByteRequestAverage"`
	RAMByteUsageAvg     float64 `json:"ramByteUsageAverage"`
	RAMByteHours        float64 `json:"ramByteHours"`
	RAMCost             float64 `json:"ramCost"`
	RAMEfficiency       float64 `json:"ramEfficiency"`

	// Totals
	TotalCost       float64 `json:"totalCost"`
	TotalEfficiency float64 `json:"totalEfficiency"`

	// Counts
	PodCount int `json:"podCount,omitempty"`
}

// AllocationProps contains properties for an allocation
type AllocationProps struct {
	Cluster        string            `json:"cluster,omitempty"`
	Node           string            `json:"node,omitempty"`
	Namespace      string            `json:"namespace,omitempty"`
	Pod            string            `json:"pod,omitempty"`
	Container      string            `json:"container,omitempty"`
	Controller     string            `json:"controller,omitempty"`
	ControllerKind string            `json:"controllerKind,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
}

// TimeWindow represents a time range
type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// AllocationResponse is the full response for the allocation API (OpenCost-compatible)
type AllocationResponse struct {
	Code   int              `json:"code"`
	Status string           `json:"status"`
	Data   []AllocationSet  `json:"data"`
}

// AllocationSet represents a set of allocations for a time period
type AllocationSet struct {
	Allocations map[string]*Allocation `json:"allocations"`
	Window      TimeWindow             `json:"window"`
	TotalCost   float64                `json:"totalCost"`
	IdleCost    float64                `json:"idleCost,omitempty"`
}

// Cost constants (configurable in production)
const (
	DefaultCPUCostPerCoreHour = 0.031611  // $/core-hour (approximate on-demand)
	DefaultRAMCostPerGBHour   = 0.004237  // $/GB-hour
)

// GetAllocations returns cost allocations based on the provided parameters
func (s *AllocationService) GetAllocations(ctx context.Context, tenantID int64, params AllocationParams) (*AllocationResponse, error) {
	// Set defaults
	if params.Limit <= 0 {
		params.Limit = 1000
	}
	if params.Accumulate == "" {
		params.Accumulate = "true"
	}

	// Parse window into start/end times
	startTime, endTime, err := s.parseWindow(params.Window)
	if err != nil {
		return nil, fmt.Errorf("invalid window: %w", err)
	}

	// Determine time steps based on accumulate parameter
	steps := s.calculateSteps(startTime, endTime, params.Step, params.Accumulate)

	// Build response with allocation sets for each step
	var allocationSets []AllocationSet

	for _, step := range steps {
		// Query allocations for this time step
		allocations, err := s.queryAllocations(ctx, tenantID, step.Start, step.End, params)
		if err != nil {
			return nil, err
		}

		// Calculate idle costs if requested
		var idleCost float64
		if params.Idle {
			idleCost, _ = s.calculateIdleCost(ctx, tenantID, step.Start, step.End)

			// Distribute idle costs if shareIdle is set
			if params.ShareIdle == "true" || params.ShareIdle == "weighted" {
				s.distributeIdleCost(allocations, idleCost, params.ShareIdle)
				idleCost = 0 // Idle cost is now distributed
			} else if params.Idle {
				// Add __idle__ allocation
				allocations["__idle__"] = &Allocation{
					Name:      "__idle__",
					Window:    TimeWindow{Start: step.Start, End: step.End},
					Start:     step.Start,
					End:       step.End,
					Minutes:   step.End.Sub(step.Start).Minutes(),
					TotalCost: idleCost,
				}
			}
		}

		// Calculate total cost
		var totalCost float64
		for _, alloc := range allocations {
			totalCost += alloc.TotalCost
		}

		allocationSets = append(allocationSets, AllocationSet{
			Allocations: allocations,
			Window:      TimeWindow{Start: step.Start, End: step.End},
			TotalCost:   totalCost,
			IdleCost:    idleCost,
		})
	}

	// If accumulate=true, merge all sets into one
	if params.Accumulate == "true" && len(allocationSets) > 1 {
		allocationSets = []AllocationSet{s.mergeAllocationSets(allocationSets, startTime, endTime)}
	}

	// Apply pagination
	allocationSets = s.paginateResults(allocationSets, params.Offset, params.Limit)

	return &AllocationResponse{
		Code:   200,
		Status: "success",
		Data:   allocationSets,
	}, nil
}

// parseWindow parses various window formats into start/end times
func (s *AllocationService) parseWindow(window string) (time.Time, time.Time, error) {
	now := time.Now().UTC()

	// Handle empty window - default to last 24 hours
	if window == "" {
		return now.Add(-24 * time.Hour), now, nil
	}

	// Handle duration format: "24h", "7d", "30d", "2w"
	durationRegex := regexp.MustCompile(`^(\d+)(m|h|d|w)$`)
	if matches := durationRegex.FindStringSubmatch(window); matches != nil {
		value, _ := strconv.Atoi(matches[1])
		unit := matches[2]

		var duration time.Duration
		switch unit {
		case "m":
			duration = time.Duration(value) * time.Minute
		case "h":
			duration = time.Duration(value) * time.Hour
		case "d":
			duration = time.Duration(value) * 24 * time.Hour
		case "w":
			duration = time.Duration(value) * 7 * 24 * time.Hour
		}
		return now.Add(-duration), now, nil
	}

	// Handle named windows
	switch window {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return start, now, nil
	case "yesterday":
		start := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, time.UTC)
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return start, end, nil
	case "week", "thisweek":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(now.Year(), now.Month(), now.Day()-(weekday-1), 0, 0, 0, 0, time.UTC)
		return start, now, nil
	case "lastweek":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		end := time.Date(now.Year(), now.Month(), now.Day()-(weekday-1), 0, 0, 0, 0, time.UTC)
		start := end.AddDate(0, 0, -7)
		return start, end, nil
	case "month", "thismonth":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, now, nil
	case "lastmonth":
		end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		start := end.AddDate(0, -1, 0)
		return start, end, nil
	}

	// Handle RFC3339 date range format: "2024-01-01T00:00:00Z,2024-01-07T23:59:59Z"
	if strings.Contains(window, ",") {
		parts := strings.Split(window, ",")
		if len(parts) == 2 {
			// Try RFC3339 first
			start, err1 := time.Parse(time.RFC3339, strings.TrimSpace(parts[0]))
			end, err2 := time.Parse(time.RFC3339, strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil {
				return start, end, nil
			}

			// Try date-only format
			start, err1 = time.Parse("2006-01-02", strings.TrimSpace(parts[0]))
			end, err2 = time.Parse("2006-01-02", strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil {
				return start, end.Add(24*time.Hour - time.Second), nil
			}

			// Try Unix timestamps
			startUnix, err1 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
			endUnix, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
			if err1 == nil && err2 == nil {
				return time.Unix(startUnix, 0).UTC(), time.Unix(endUnix, 0).UTC(), nil
			}
		}
	}

	return time.Time{}, time.Time{}, fmt.Errorf("unrecognized window format: %s", window)
}

// calculateSteps determines time intervals based on step and accumulate parameters
func (s *AllocationService) calculateSteps(start, end time.Time, step, accumulate string) []TimeWindow {
	// If accumulate=true or no step specified, return single window
	if accumulate == "true" || (step == "" && accumulate == "") {
		return []TimeWindow{{Start: start, End: end}}
	}

	// Parse step duration
	var stepDuration time.Duration
	if step != "" {
		stepDuration = s.parseStepDuration(step)
	} else {
		// Use accumulate value as step
		stepDuration = s.parseStepDuration(accumulate)
	}

	if stepDuration == 0 {
		return []TimeWindow{{Start: start, End: end}}
	}

	// Generate steps
	var steps []TimeWindow
	current := start
	for current.Before(end) {
		stepEnd := current.Add(stepDuration)
		if stepEnd.After(end) {
			stepEnd = end
		}
		steps = append(steps, TimeWindow{Start: current, End: stepEnd})
		current = stepEnd
	}

	return steps
}

func (s *AllocationService) parseStepDuration(step string) time.Duration {
	durationRegex := regexp.MustCompile(`^(\d+)(m|h|d|w)$`)
	if matches := durationRegex.FindStringSubmatch(step); matches != nil {
		value, _ := strconv.Atoi(matches[1])
		unit := matches[2]
		switch unit {
		case "m":
			return time.Duration(value) * time.Minute
		case "h":
			return time.Duration(value) * time.Hour
		case "d":
			return time.Duration(value) * 24 * time.Hour
		case "w":
			return time.Duration(value) * 7 * 24 * time.Hour
		}
	}

	// Handle named intervals
	switch step {
	case "hour", "hourly":
		return time.Hour
	case "day", "daily":
		return 24 * time.Hour
	case "week", "weekly":
		return 7 * 24 * time.Hour
	}

	return 0
}

// queryAllocations executes the allocation query based on aggregation type
func (s *AllocationService) queryAllocations(ctx context.Context, tenantID int64, startTime, endTime time.Time, params AllocationParams) (map[string]*Allocation, error) {
	// Parse aggregate parameter (supports comma-separated multi-aggregation)
	aggregates := strings.Split(params.Aggregate, ",")
	if len(aggregates) == 0 || params.Aggregate == "" {
		aggregates = []string{"namespace"}
	}

	// Build grouping columns
	var groupByCols, selectCols []string
	var labelKeys []string

	for _, agg := range aggregates {
		agg = strings.TrimSpace(strings.ToLower(agg))

		if strings.HasPrefix(agg, "label:") {
			labelKey := strings.TrimPrefix(agg, "label:")
			labelKeys = append(labelKeys, labelKey)
			selectCols = append(selectCols, fmt.Sprintf("COALESCE(labels->>'%s', '__unallocated__')", labelKey))
			groupByCols = append(groupByCols, fmt.Sprintf("labels->>'%s'", labelKey))
		} else {
			switch agg {
			case "cluster":
				selectCols = append(selectCols, "cluster_name")
				groupByCols = append(groupByCols, "cluster_name")
			case "namespace":
				selectCols = append(selectCols, "namespace")
				groupByCols = append(groupByCols, "namespace")
			case "node":
				selectCols = append(selectCols, "node_name")
				groupByCols = append(groupByCols, "node_name")
			case "pod":
				selectCols = append(selectCols, "CONCAT(namespace, '/', pod_name)")
				groupByCols = append(groupByCols, "namespace", "pod_name")
			case "controller":
				// Extract controller from pod name (remove hash suffix)
				selectCols = append(selectCols, "REGEXP_REPLACE(pod_name, '-[a-z0-9]{5,10}(-[a-z0-9]{5})?$', '')")
				groupByCols = append(groupByCols, "REGEXP_REPLACE(pod_name, '-[a-z0-9]{5,10}(-[a-z0-9]{5})?$', '')")
			default:
				selectCols = append(selectCols, "namespace")
				groupByCols = append(groupByCols, "namespace")
			}
		}
	}

	// Build name expression (concatenate multiple aggregations)
	var nameExpr string
	if len(selectCols) == 1 {
		nameExpr = selectCols[0]
	} else {
		nameExpr = fmt.Sprintf("CONCAT(%s)", strings.Join(selectCols, ", '/', "))
	}

	// Build query
	query := fmt.Sprintf(`
		SELECT
			%s as name,
			cluster_name,
			namespace,
			node_name,
			AVG(cpu_millicores) / 1000.0 as cpu_cores_usage,
			AVG(cpu_request_millicores) / 1000.0 as cpu_cores_request,
			AVG(memory_bytes) as memory_bytes_usage,
			AVG(memory_request_bytes) as memory_bytes_request,
			COUNT(DISTINCT pod_name) as pod_count
		FROM pod_metrics
		WHERE tenant_id = $1
			AND time >= $2
			AND time <= $3
			AND pod_name != '__aggregate__'
	`, nameExpr)

	args := []interface{}{tenantID, startTime, endTime}
	argIdx := 4

	// Add label existence filters for label aggregations
	for _, labelKey := range labelKeys {
		query += fmt.Sprintf(" AND labels ? '%s'", labelKey)
	}

	// Add filters
	for _, filter := range params.Filters {
		// Parse filter: "namespace:value", "cluster:value", "label:key=value"
		parts := strings.SplitN(filter, ":", 2)
		if len(parts) != 2 {
			continue
		}
		filterType := strings.ToLower(parts[0])
		filterValue := parts[1]

		switch filterType {
		case "namespace":
			values := strings.Split(filterValue, ",")
			if len(values) == 1 {
				query += fmt.Sprintf(" AND namespace = $%d", argIdx)
				args = append(args, filterValue)
				argIdx++
			} else {
				placeholders := make([]string, len(values))
				for i, v := range values {
					placeholders[i] = fmt.Sprintf("$%d", argIdx)
					args = append(args, strings.TrimSpace(v))
					argIdx++
				}
				query += fmt.Sprintf(" AND namespace IN (%s)", strings.Join(placeholders, ","))
			}
		case "cluster":
			values := strings.Split(filterValue, ",")
			if len(values) == 1 {
				query += fmt.Sprintf(" AND cluster_name = $%d", argIdx)
				args = append(args, filterValue)
				argIdx++
			} else {
				placeholders := make([]string, len(values))
				for i, v := range values {
					placeholders[i] = fmt.Sprintf("$%d", argIdx)
					args = append(args, strings.TrimSpace(v))
					argIdx++
				}
				query += fmt.Sprintf(" AND cluster_name IN (%s)", strings.Join(placeholders, ","))
			}
		case "node":
			query += fmt.Sprintf(" AND node_name = $%d", argIdx)
			args = append(args, filterValue)
			argIdx++
		case "label":
			// label filter format: "label:key=value"
			labelParts := strings.SplitN(filterValue, "=", 2)
			if len(labelParts) == 2 {
				query += fmt.Sprintf(" AND labels->>'%s' = $%d", labelParts[0], argIdx)
				args = append(args, labelParts[1])
				argIdx++
			}
		case "pod":
			query += fmt.Sprintf(" AND pod_name LIKE $%d", argIdx)
			args = append(args, "%"+filterValue+"%")
			argIdx++
		}
	}

	query += fmt.Sprintf(`
		GROUP BY %s, cluster_name, namespace, node_name
		ORDER BY cpu_cores_usage DESC
	`, strings.Join(groupByCols, ", "))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	results := make(map[string]*Allocation)
	durationHours := endTime.Sub(startTime).Hours()
	if durationHours <= 0 {
		durationHours = 1
	}
	minutes := endTime.Sub(startTime).Minutes()

	for rows.Next() {
		var name, clusterName, namespace, nodeName string
		var cpuCoresUsage, cpuCoresRequest, memBytesUsage, memBytesRequest float64
		var podCount int

		if err := rows.Scan(&name, &clusterName, &namespace, &nodeName,
			&cpuCoresUsage, &cpuCoresRequest, &memBytesUsage, &memBytesRequest, &podCount); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		if name == "" {
			name = "__unallocated__"
		}

		// Calculate costs using OpenCost model: max(request, usage)
		effectiveCPU := cpuCoresRequest
		if cpuCoresUsage > cpuCoresRequest {
			effectiveCPU = cpuCoresUsage
		}
		effectiveRAM := memBytesRequest
		if memBytesUsage > memBytesRequest {
			effectiveRAM = memBytesUsage
		}

		cpuCoreHours := effectiveCPU * durationHours
		ramByteHours := effectiveRAM * durationHours

		// Get pricing rates (dynamic or default)
		cpuRate, memRate, _ := s.getNodePricing(ctx, tenantID, clusterName, nodeName, startTime)
		cpuCost := cpuCoreHours * cpuRate
		ramCost := (ramByteHours / 1024 / 1024 / 1024) * memRate
		totalCost := cpuCost + ramCost

		// Calculate efficiencies
		var cpuEfficiency, ramEfficiency float64
		if cpuCoresRequest > 0 {
			cpuEfficiency = cpuCoresUsage / cpuCoresRequest
		}
		if memBytesRequest > 0 {
			ramEfficiency = memBytesUsage / memBytesRequest
		}
		totalEfficiency := (cpuEfficiency + ramEfficiency) / 2

		alloc := &Allocation{
			Name:   name,
			Window: TimeWindow{Start: startTime, End: endTime},
			Start:  startTime,
			End:    endTime,
			Minutes: minutes,

			CPUCores:          effectiveCPU,
			CPUCoreRequestAvg: cpuCoresRequest,
			CPUCoreUsageAvg:   cpuCoresUsage,
			CPUCoreHours:      cpuCoreHours,
			CPUCost:           cpuCost,
			CPUEfficiency:     cpuEfficiency,

			RAMBytes:          effectiveRAM,
			RAMByteRequestAvg: memBytesRequest,
			RAMByteUsageAvg:   memBytesUsage,
			RAMByteHours:      ramByteHours,
			RAMCost:           ramCost,
			RAMEfficiency:     ramEfficiency,

			TotalCost:       totalCost,
			TotalEfficiency: totalEfficiency,
			PodCount:        podCount,

			Properties: AllocationProps{
				Cluster:   clusterName,
				Namespace: namespace,
				Node:      nodeName,
			},
		}

		// Merge if same name exists (aggregate across clusters/nodes)
		if existing, ok := results[name]; ok {
			existing.CPUCores += alloc.CPUCores
			existing.CPUCoreHours += alloc.CPUCoreHours
			existing.CPUCost += alloc.CPUCost
			existing.RAMBytes += alloc.RAMBytes
			existing.RAMByteHours += alloc.RAMByteHours
			existing.RAMCost += alloc.RAMCost
			existing.TotalCost += alloc.TotalCost
			existing.PodCount += alloc.PodCount
		} else {
			results[name] = alloc
		}
	}

	return results, rows.Err()
}

// calculateIdleCost calculates the cost of unused cluster capacity
func (s *AllocationService) calculateIdleCost(ctx context.Context, tenantID int64, startTime, endTime time.Time) (float64, error) {
	query := `
		WITH node_capacity AS (
			SELECT
				cluster_name,
				AVG(cpu_capacity) as avg_cpu_capacity,
				AVG(memory_capacity) as avg_memory_capacity,
				AVG(hourly_cost_usd) as avg_hourly_cost
			FROM node_metrics
			WHERE tenant_id = $1 AND time >= $2 AND time <= $3
			GROUP BY cluster_name
		),
		pod_usage AS (
			SELECT
				cluster_name,
				SUM(cpu_millicores) / 1000.0 as total_cpu_used,
				SUM(memory_bytes) as total_memory_used
			FROM pod_metrics
			WHERE tenant_id = $1 AND time >= $2 AND time <= $3
				AND pod_name != '__aggregate__'
			GROUP BY cluster_name
		)
		SELECT
			COALESCE(SUM(
				CASE
					WHEN nc.avg_cpu_capacity > 0
					THEN (1 - LEAST(COALESCE(pu.total_cpu_used, 0) / nc.avg_cpu_capacity, 1)) * nc.avg_hourly_cost
					ELSE 0
				END
			), 0) as idle_cost_per_hour
		FROM node_capacity nc
		LEFT JOIN pod_usage pu ON nc.cluster_name = pu.cluster_name
	`

	var idleCostPerHour float64
	err := s.pool.QueryRow(ctx, query, tenantID, startTime, endTime).Scan(&idleCostPerHour)
	if err != nil {
		return 0, err
	}

	duration := endTime.Sub(startTime).Hours()
	if duration <= 0 {
		duration = 1
	}

	return idleCostPerHour * duration, nil
}

// distributeIdleCost distributes idle costs across allocations
func (s *AllocationService) distributeIdleCost(allocations map[string]*Allocation, idleCost float64, method string) {
	if idleCost <= 0 || len(allocations) == 0 {
		return
	}

	if method == "weighted" {
		// Distribute proportionally based on total cost
		var totalCost float64
		for _, alloc := range allocations {
			totalCost += alloc.TotalCost
		}
		if totalCost <= 0 {
			return
		}
		for _, alloc := range allocations {
			share := (alloc.TotalCost / totalCost) * idleCost
			alloc.TotalCost += share
		}
	} else {
		// Distribute evenly
		share := idleCost / float64(len(allocations))
		for _, alloc := range allocations {
			alloc.TotalCost += share
		}
	}
}

// mergeAllocationSets merges multiple allocation sets into one
func (s *AllocationService) mergeAllocationSets(sets []AllocationSet, start, end time.Time) AllocationSet {
	merged := AllocationSet{
		Allocations: make(map[string]*Allocation),
		Window:      TimeWindow{Start: start, End: end},
	}

	for _, set := range sets {
		merged.IdleCost += set.IdleCost
		for name, alloc := range set.Allocations {
			if existing, ok := merged.Allocations[name]; ok {
				existing.CPUCores += alloc.CPUCores
				existing.CPUCoreHours += alloc.CPUCoreHours
				existing.CPUCost += alloc.CPUCost
				existing.RAMBytes += alloc.RAMBytes
				existing.RAMByteHours += alloc.RAMByteHours
				existing.RAMCost += alloc.RAMCost
				existing.TotalCost += alloc.TotalCost
				existing.PodCount += alloc.PodCount
				existing.Minutes += alloc.Minutes
			} else {
				allocCopy := *alloc
				allocCopy.Window = TimeWindow{Start: start, End: end}
				allocCopy.Start = start
				allocCopy.End = end
				merged.Allocations[name] = &allocCopy
			}
		}
	}

	// Recalculate total cost
	for _, alloc := range merged.Allocations {
		merged.TotalCost += alloc.TotalCost
	}

	return merged
}

// paginateResults applies offset/limit pagination to allocation sets
func (s *AllocationService) paginateResults(sets []AllocationSet, offset, limit int) []AllocationSet {
	if offset <= 0 && limit <= 0 {
		return sets
	}

	for i := range sets {
		// Convert map to sorted slice for consistent pagination
		names := make([]string, 0, len(sets[i].Allocations))
		for name := range sets[i].Allocations {
			names = append(names, name)
		}
		sort.Strings(names)

		// Apply offset and limit
		start := offset
		if start > len(names) {
			start = len(names)
		}
		end := start + limit
		if end > len(names) || limit <= 0 {
			end = len(names)
		}

		// Create new map with paginated results
		paginated := make(map[string]*Allocation)
		for _, name := range names[start:end] {
			paginated[name] = sets[i].Allocations[name]
		}
		sets[i].Allocations = paginated
	}

	return sets
}
