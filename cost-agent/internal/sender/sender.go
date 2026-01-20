package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bugfreev587/cost-agent/internal/collector"
	"github.com/cenkalti/backoff/v4"
)

// Payloads
type NamespaceCostData struct {
	Namespace          string  `json:"namespace"`
	PodCount           int     `json:"pod_count"`
	TotalCPUMillicores int64   `json:"total_cpu_millicores"`
	TotalMemoryBytes   int64   `json:"total_memory_bytes"`
	EstimatedCostUSD   float64 `json:"estimated_cost_usd"`
}

// ContainerMetricData represents metrics for a single container
type ContainerMetricData struct {
	ContainerName        string `json:"container_name"`
	CPUUsageMillicores   int64  `json:"cpu_usage_millicores"`
	MemoryUsageBytes     int64  `json:"memory_usage_bytes"`
	CPURequestMillicores int64  `json:"cpu_request_millicores"`
	MemoryRequestBytes   int64  `json:"memory_request_bytes"`
	CPULimitMillicores   int64  `json:"cpu_limit_millicores,omitempty"`
	MemoryLimitBytes     int64  `json:"memory_limit_bytes,omitempty"`
}

type PodMetricData struct {
	PodName              string                 `json:"pod_name"`
	Namespace            string                 `json:"namespace"`
	NodeName             string                 `json:"node_name"`
	CPUUsageMillicores   int64                  `json:"cpu_usage_millicores"`
	MemoryUsageBytes     int64                  `json:"memory_usage_bytes"`
	CPURequestMillicores int64                  `json:"cpu_request_millicores"`
	MemoryRequestBytes   int64                  `json:"memory_request_bytes"`
	CPULimitMillicores   int64                  `json:"cpu_limit_millicores,omitempty"`
	MemoryLimitBytes     int64                  `json:"memory_limit_bytes,omitempty"`
	// New Priority 1 fields
	Labels     map[string]string      `json:"labels,omitempty"`
	Phase      string                 `json:"phase,omitempty"`
	QoSClass   string                 `json:"qos_class,omitempty"`
	Containers []ContainerMetricData  `json:"containers,omitempty"`
}

type AgentMetricsPayload struct {
	ClusterName    string                       `json:"cluster_name"`
	Timestamp      int64                        `json:"timestamp"`
	PodMetrics     []PodMetricData              `json:"pod_metrics"`
	NamespaceCosts map[string]NamespaceCostData `json:"namespace_costs"`
	NodeMetrics    []collector.NodeMetric       `json:"node_metrics"`
}

type Sender struct {
	Client    *http.Client
	ServerURL string
	APIKey    string
}

func NewSender(serverURL, apiKey string, timeout time.Duration) *Sender {
	return &Sender{
		Client:    &http.Client{Timeout: timeout},
		ServerURL: serverURL,
		APIKey:    apiKey,
	}
}

func (s *Sender) Send(ctx context.Context, payload AgentMetricsPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.ServerURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "ApiKey "+s.APIKey)

	// exponential backoff for transient errors
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 2 * time.Minute

	operation := func() error {
		resp, err := s.Client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			io.Copy(io.Discard, resp.Body)
			return nil
		}
		// treat 4xx as permanent (except 429)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
			b, _ := io.ReadAll(resp.Body)
			return backoff.Permanent(fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(b)))
		}
		// otherwise retry
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(b))
	}

	if err := backoff.Retry(operation, bo); err != nil {
		return fmt.Errorf("send failed: %w", err)
	}
	return nil
}
