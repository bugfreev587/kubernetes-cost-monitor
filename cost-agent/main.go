package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bugfreev587/cost-agent/internal/collector"
	"github.com/bugfreev587/cost-agent/internal/config"
	"github.com/bugfreev587/cost-agent/internal/sender"
)

func main() {
	// Determine config file path
	// If AGENT_CONFIG_FILE is set, use it; otherwise use empty string to skip config file
	configPath := os.Getenv("AGENT_CONFIG_FILE")
	// If not set, config.Load will use environment variables only (recommended for Kubernetes)
	log.Printf("configPath: %s", configPath)
	// load config
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if cfg.APIKey == "" {
		log.Fatal("API key not provided. Set AGENT_API_KEY, API_KEY, or set api_key in config file")
	}

	// create collector
	col, err := collector.NewCollector(cfg.UseMetricsAPI, cfg.ClusterName, cfg.NamespaceFilter, cfg.CollectPodLabels, cfg.CollectContainerMetrics)
	if err != nil {
		log.Fatalf("collector init: %v", err)
	}
	log.Printf("collector initialized (collectLabels=%v, collectContainers=%v)", cfg.CollectPodLabels, cfg.CollectContainerMetrics)
	// create sender
	s := sender.NewSender(cfg.ServerURL, cfg.APIKey, cfg.HTTPTimeout)
	log.Printf("sender created")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(cfg.CollectInterval)
	defer ticker.Stop()
	log.Printf("ticker created")
	// initial immediate collect
	go func() {
		if err := collectAndSend(ctx, col, s, cfg.HTTPTimeout); err != nil {
			log.Printf("initial collect send error: %v", err)
		}
	}()

	for {
		select {
		case <-ticker.C:
			log.Println("collecting and sending metrics")
			if err := collectAndSend(ctx, col, s, cfg.HTTPTimeout); err != nil {
				log.Printf("collect send error: %v", err)
			}
			log.Printf("metrics collected and sent, sleeping for %v seconds", cfg.CollectInterval)
		case <-stop:
			log.Println("shutting down agent")
			cancel()
			time.Sleep(1 * time.Second)
			return
		}
	}
}

// convertContainers converts collector.ContainerMetric to sender.ContainerMetricData
func convertContainers(containers []collector.ContainerMetric) []sender.ContainerMetricData {
	if containers == nil {
		return nil
	}
	result := make([]sender.ContainerMetricData, len(containers))
	for i, c := range containers {
		result[i] = sender.ContainerMetricData{
			ContainerName:        c.ContainerName,
			CPUUsageMillicores:   c.CPUUsageMillicores,
			MemoryUsageBytes:     c.MemoryUsageBytes,
			CPURequestMillicores: c.CPURequestMillicores,
			MemoryRequestBytes:   c.MemoryRequestBytes,
			CPULimitMillicores:   c.CPULimitMillicores,
			MemoryLimitBytes:     c.MemoryLimitBytes,
		}
	}
	return result
}

func collectAndSend(ctx context.Context, c *collector.Collector, s *sender.Sender, httpTimeout time.Duration) error {
	// Use httpTimeout for context, with extra buffer for collection
	timeout := httpTimeout + 30*time.Second
	ctx2, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	// collect
	pods, err := c.CollectPodMetrics(ctx2)
	if err != nil {
		// non-fatal, continue with what we have
		log.Printf("collect pods error: %v", err)
	}
	nodes, err := c.CollectNodeMetrics(ctx2)
	if err != nil {
		log.Printf("collect nodes error: %v", err)
	}
	// aggregate
	aggs := collector.AggregateByNamespace(pods)
	// map to sender payload
	payload := sender.AgentMetricsPayload{
		ClusterName:    c.ClusterName,
		Timestamp:      time.Now().Unix(),
		PodMetrics:     []sender.PodMetricData{},
		NamespaceCosts: map[string]sender.NamespaceCostData{},
		NodeMetrics:    nodes,
	}
	// Add individual pod metrics
	for _, p := range pods {
		podData := sender.PodMetricData{
			PodName:              p.PodName,
			Namespace:            p.Namespace,
			NodeName:             p.NodeName,
			CPUUsageMillicores:   p.CPUUsageMillicores,
			MemoryUsageBytes:     p.MemoryUsageBytes,
			CPURequestMillicores: p.CPURequestMillicores,
			MemoryRequestBytes:   p.MemoryRequestBytes,
			CPULimitMillicores:   p.CPULimitMillicores,
			MemoryLimitBytes:     p.MemoryLimitBytes,
			// New Priority 1 fields
			Labels:     p.Labels,
			Phase:      p.Phase,
			QoSClass:   p.QoSClass,
			Containers: convertContainers(p.Containers),
		}
		payload.PodMetrics = append(payload.PodMetrics, podData)
	}
	// Add namespace aggregates for backward compatibility
	for _, a := range aggs {
		payload.NamespaceCosts[a.Namespace] = sender.NamespaceCostData{
			Namespace:          a.Namespace,
			PodCount:           a.PodCount,
			TotalCPUMillicores: a.TotalCPUmilli,
			TotalMemoryBytes:   a.TotalMemBytes,
			EstimatedCostUSD:   0.0, // server will compute actual cost if needed
		}
	}
	// send
	return s.Send(ctx2, payload)
}
