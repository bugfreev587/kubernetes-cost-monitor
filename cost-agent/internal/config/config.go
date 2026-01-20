package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	ServerURL              string        `mapstructure:"server_url" yaml:"server_url"`
	APIKey                 string        `mapstructure:"api_key" yaml:"api_key"`
	ClusterName            string        `mapstructure:"cluster_name" yaml:"cluster_name"`
	CollectInterval        time.Duration `mapstructure:"collect_interval" yaml:"collect_interval"`
	HTTPTimeout            time.Duration `mapstructure:"http_timeout" yaml:"http_timeout"`
	UseMetricsAPI          bool          `mapstructure:"use_metrics_api" yaml:"use_metrics_api"`
	NamespaceFilter        string        `mapstructure:"namespace_filter" yaml:"namespace_filter"` // optional: only collect this namespace if set
	CollectPodLabels       bool          `mapstructure:"collect_pod_labels" yaml:"collect_pod_labels"`
	CollectContainerMetrics bool         `mapstructure:"collect_container_metrics" yaml:"collect_container_metrics"`
}

// Load loads configuration from a YAML file path with environment variable overrides
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set environment variable prefix
	v.SetEnvPrefix("AGENT")
	v.AutomaticEnv()

	// Set config file path if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
	}

	// Set defaults (will be overridden by config file or env vars)
	v.SetDefault("server_url", "http://host.docker.internal:8080")
	v.SetDefault("cluster_name", "cost-dashboard-dev")
	v.SetDefault("collect_interval", 600) // seconds (10 minutes)
	v.SetDefault("http_timeout", 10)      // seconds
	v.SetDefault("use_metrics_api", true)
	v.SetDefault("namespace_filter", "")
	v.SetDefault("collect_pod_labels", true)        // Enable by default
	v.SetDefault("collect_container_metrics", true) // Enable by default

	// Load values directly and convert durations manually
	// Viper doesn't automatically convert int to Duration for YAML files
	cfg := Config{
		ServerURL:              v.GetString("server_url"),
		APIKey:                 v.GetString("api_key"),
		ClusterName:            v.GetString("cluster_name"),
		CollectInterval:        time.Duration(v.GetInt("collect_interval")) * time.Second,
		HTTPTimeout:            time.Duration(v.GetInt("http_timeout")) * time.Second,
		UseMetricsAPI:          v.GetBool("use_metrics_api"),
		NamespaceFilter:        v.GetString("namespace_filter"),
		CollectPodLabels:       v.GetBool("collect_pod_labels"),
		CollectContainerMetrics: v.GetBool("collect_container_metrics"),
	}

	// Allow API key to be set via environment variable (AGENT_API_KEY or API_KEY)
	if cfg.APIKey == "" {
		if apiKey := os.Getenv("AGENT_API_KEY"); apiKey != "" {
			cfg.APIKey = apiKey
		} else if apiKey := os.Getenv("API_KEY"); apiKey != "" {
			cfg.APIKey = apiKey
		}
	}

	return &cfg, nil
}
