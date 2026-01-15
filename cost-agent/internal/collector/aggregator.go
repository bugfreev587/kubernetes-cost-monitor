package collector

import "time"

// NamespaceAggregate holds aggregated metrics per namespace
type NamespaceAggregate struct {
	Timestamp     time.Time
	ClusterName   string
	Namespace     string
	TotalCPUmilli int64
	TotalMemBytes int64
	PodCount      int
}

func AggregateByNamespace(pods []PodMetric) []NamespaceAggregate {
	m := map[string]*NamespaceAggregate{}
	for _, p := range pods {
		key := p.Namespace
		if _, ok := m[key]; !ok {
			m[key] = &NamespaceAggregate{
				Timestamp:   p.Timestamp,
				ClusterName: p.ClusterName,
				Namespace:   p.Namespace,
			}
		}
		a := m[key]
		a.TotalCPUmilli += max64(p.CPUUsageMillicores, p.CPURequestMillicores)
		a.TotalMemBytes += max64(p.MemoryUsageBytes, p.MemoryRequestBytes)
		a.PodCount++
	}
	out := []NamespaceAggregate{}
	for _, v := range m {
		out = append(out, *v)
	}
	return out
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
