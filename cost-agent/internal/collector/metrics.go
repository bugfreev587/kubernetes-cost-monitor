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
	K8sClient       *kubernetes.Clientset
	MetricsClient   *metricsv.Clientset
	ClusterName     string
	UseMetricsAPI   bool
	NamespaceFilter string
}

// NewCollector creates a collector using in-cluster config or kubeconfig if KUBECONFIG provided.
func NewCollector(useMetricsAPI bool, clusterName, namespaceFilter string) (*Collector, error) {
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
		K8sClient:       kc,
		MetricsClient:   mc,
		ClusterName:     clusterName,
		UseMetricsAPI:   useMetricsAPI && mc != nil,
		NamespaceFilter: namespaceFilter,
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
		for _, cs := range p.Spec.Containers {
			if q, ok := cs.Resources.Requests[v1.ResourceCPU]; ok {
				cpuReq += q.MilliValue()
			}
			if q, ok := cs.Resources.Requests[v1.ResourceMemory]; ok {
				memReq += q.Value()
			}
			if q, ok := cs.Resources.Limits[v1.ResourceCPU]; ok {
				cpuLimit += q.MilliValue()
			}
			if q, ok := cs.Resources.Limits[v1.ResourceMemory]; ok {
				memLimit += q.Value()
			}
		}
		key := fmt.Sprintf("%s/%s", p.Namespace, p.Name)
		requestsMap[key] = PodMetric{
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
	}

	// if metrics API available, fetch actual usage
	if c.UseMetricsAPI && c.MetricsClient != nil {
		podMetricsList, err := c.MetricsClient.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, pm := range podMetricsList.Items {
				if c.NamespaceFilter != "" && pm.Namespace != c.NamespaceFilter {
					continue
				}
				for _, ctn := range pm.Containers {
					// sum container usages
					key := fmt.Sprintf("%s/%s", pm.Namespace, pm.Name)
					pmEntry := requestsMap[key]
					cpuQty := ctn.Usage.Cpu()
					memQty := ctn.Usage.Memory()
					if cpuQty != nil {
						pmEntry.CPUUsageMillicores += cpuQty.MilliValue()
					}
					if memQty != nil {
						pmEntry.MemoryUsageBytes += memQty.Value()
					}
					requestsMap[key] = pmEntry
				}
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
