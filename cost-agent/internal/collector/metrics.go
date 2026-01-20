package collector

import (
	"context"
	"fmt"
	"os"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// ContainerMetric represents metrics for a single container within a pod
type ContainerMetric struct {
	ContainerName        string `json:"container_name"`
	CPUUsageMillicores   int64  `json:"cpu_usage_millicores"`
	MemoryUsageBytes     int64  `json:"memory_usage_bytes"`
	CPURequestMillicores int64  `json:"cpu_request_millicores"`
	MemoryRequestBytes   int64  `json:"memory_request_bytes"`
	CPULimitMillicores   int64  `json:"cpu_limit_millicores"`
	MemoryLimitBytes     int64  `json:"memory_limit_bytes"`
}

type PodMetric struct {
	Timestamp            time.Time
	ClusterName          string
	Namespace            string
	PodName              string
	NodeName             string
	CPUUsageMillicores   int64
	MemoryUsageBytes     int64
	CPURequestMillicores int64
	MemoryRequestBytes   int64
	CPULimitMillicores   int64
	MemoryLimitBytes     int64
	// New fields for Priority 1 improvements
	Labels     map[string]string  `json:"labels,omitempty"`     // Pod labels for cost allocation
	Phase      string             `json:"phase,omitempty"`      // Running, Pending, Succeeded, Failed, Unknown
	QoSClass   string             `json:"qos_class,omitempty"`  // Guaranteed, Burstable, BestEffort
	Containers []ContainerMetric  `json:"containers,omitempty"` // Per-container breakdown
}

type NodeMetric struct {
	Timestamp         time.Time
	ClusterName       string
	NodeName          string
	InstanceType      string
	CPUCapacity       int64
	MemoryCapacity    int64
	CPUAllocatable    int64
	MemoryAllocatable int64
}

type Collector struct {
	K8sClient              *kubernetes.Clientset
	MetricsClient          *metricsv.Clientset
	ClusterName            string
	UseMetricsAPI          bool
	NamespaceFilter        string
	CollectPodLabels       bool
	CollectContainerMetrics bool
}

// NewCollector creates a collector using in-cluster config or kubeconfig if KUBECONFIG provided.
func NewCollector(useMetricsAPI bool, clusterName, namespaceFilter string, collectPodLabels, collectContainerMetrics bool) (*Collector, error) {
	var cfg *rest.Config
	var err error
	if kube := os.Getenv("KUBECONFIG"); kube != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kube)
	} else {
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, err
	}

	kc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	var mc *metricsv.Clientset
	if useMetricsAPI {
		mc, _ = metricsv.NewForConfig(cfg) // may be nil if not available
	}

	return &Collector{
		K8sClient:              kc,
		MetricsClient:          mc,
		ClusterName:            clusterName,
		UseMetricsAPI:          useMetricsAPI && mc != nil,
		NamespaceFilter:        namespaceFilter,
		CollectPodLabels:       collectPodLabels,
		CollectContainerMetrics: collectContainerMetrics,
	}, nil
}

// CollectPodMetrics collects pod-level metrics. If metrics API unavailable, fall back to requests.
func (c *Collector) CollectPodMetrics(ctx context.Context) ([]PodMetric, error) {
	res := []PodMetric{}
	// list pods
	pods, err := c.K8sClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// map to requests
	requestsMap := map[string]PodMetric{} // key namespace/pod
	for _, p := range pods.Items {
		if c.NamespaceFilter != "" && p.Namespace != c.NamespaceFilter {
			continue
		}
		// calculate pod total requests and limits
		var cpuReq int64 = 0
		var memReq int64 = 0
		var cpuLimit int64 = 0
		var memLimit int64 = 0

		// Collect container-level metrics
		containers := make([]ContainerMetric, 0, len(p.Spec.Containers))
		for _, cs := range p.Spec.Containers {
			containerMetric := ContainerMetric{
				ContainerName: cs.Name,
			}

			if q, ok := cs.Resources.Requests[v1.ResourceCPU]; ok {
				containerMetric.CPURequestMillicores = q.MilliValue()
				cpuReq += q.MilliValue()
			}
			if q, ok := cs.Resources.Requests[v1.ResourceMemory]; ok {
				containerMetric.MemoryRequestBytes = q.Value()
				memReq += q.Value()
			}
			if q, ok := cs.Resources.Limits[v1.ResourceCPU]; ok {
				containerMetric.CPULimitMillicores = q.MilliValue()
				cpuLimit += q.MilliValue()
			}
			if q, ok := cs.Resources.Limits[v1.ResourceMemory]; ok {
				containerMetric.MemoryLimitBytes = q.Value()
				memLimit += q.Value()
			}

			containers = append(containers, containerMetric)
		}

		key := fmt.Sprintf("%s/%s", p.Namespace, p.Name)
		podMetric := PodMetric{
			Timestamp:            time.Now().UTC(),
			ClusterName:          c.ClusterName,
			Namespace:            p.Namespace,
			PodName:              p.Name,
			NodeName:             p.Spec.NodeName,
			CPURequestMillicores: cpuReq,
			MemoryRequestBytes:   memReq,
			CPULimitMillicores:   cpuLimit,
			MemoryLimitBytes:     memLimit,
		}

		// Only collect new Priority 1 fields if enabled
		if c.CollectPodLabels {
			podMetric.Labels = p.Labels
		}
		podMetric.Phase = string(p.Status.Phase)      // Always collect phase
		podMetric.QoSClass = string(p.Status.QOSClass) // Always collect QoS class

		if c.CollectContainerMetrics {
			podMetric.Containers = containers
		}

		requestsMap[key] = podMetric
	}

	// if metrics API available, fetch actual usage
	if c.UseMetricsAPI && c.MetricsClient != nil {
		podMetricsList, err := c.MetricsClient.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, pm := range podMetricsList.Items {
				if c.NamespaceFilter != "" && pm.Namespace != c.NamespaceFilter {
					continue
				}
				key := fmt.Sprintf("%s/%s", pm.Namespace, pm.Name)
				pmEntry := requestsMap[key]

				// Update container-level usage metrics
				for _, ctn := range pm.Containers {
					cpuQty := ctn.Usage.Cpu()
					memQty := ctn.Usage.Memory()

					// Sum pod-level usage
					if cpuQty != nil {
						pmEntry.CPUUsageMillicores += cpuQty.MilliValue()
					}
					if memQty != nil {
						pmEntry.MemoryUsageBytes += memQty.Value()
					}

					// Update container-level usage in Containers array
					for i := range pmEntry.Containers {
						if pmEntry.Containers[i].ContainerName == ctn.Name {
							if cpuQty != nil {
								pmEntry.Containers[i].CPUUsageMillicores = cpuQty.MilliValue()
							}
							if memQty != nil {
								pmEntry.Containers[i].MemoryUsageBytes = memQty.Value()
							}
							break
						}
					}
				}

				requestsMap[key] = pmEntry
			}
		}
	}

	// convert map to slice
	for _, v := range requestsMap {
		res = append(res, v)
	}
	return res, nil
}

// CollectNodeMetrics collects node capacities and allocatable
func (c *Collector) CollectNodeMetrics(ctx context.Context) ([]NodeMetric, error) {
	out := []NodeMetric{}
	nodes, err := c.K8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, n := range nodes.Items {
		cpuCap := n.Status.Capacity.Cpu()
		memCap := n.Status.Capacity.Memory()
		cpuAlloc := n.Status.Allocatable.Cpu()
		memAlloc := n.Status.Allocatable.Memory()
		nm := NodeMetric{
			Timestamp:    time.Now().UTC(),
			ClusterName:  c.ClusterName,
			NodeName:     n.Name,
			InstanceType: n.Labels["node.kubernetes.io/instance-type"],
		}
		if cpuCap != nil {
			nm.CPUCapacity = cpuCap.MilliValue()
		}
		if memCap != nil {
			nm.MemoryCapacity = memCap.Value()
		}
		if cpuAlloc != nil {
			nm.CPUAllocatable = cpuAlloc.MilliValue()
		}
		if memAlloc != nil {
			nm.MemoryAllocatable = memAlloc.Value()
		}
		out = append(out, nm)
	}
	return out, nil
}
