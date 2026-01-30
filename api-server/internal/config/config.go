package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/spf13/viper"
)

type ServerCfg struct {
	Host                string   `mapstructure:"host"`
	Port                string   `mapstructure:"port"`
	ReadTimeoutSeconds  int      `mapstructure:"read_timeout_seconds"`
	WriteTimeoutSeconds int      `mapstructure:"write_timeout_seconds"`
	CORSOrigins         []string `yaml:"cors_origins,omitempty"`
	RateLimitPerMinute  int      `yaml:"rate_limit_per_minute,omitempty"`
}

type PostgresCfg struct {
	DSN string `mapstructure:"dsn"`
}

type RedisCfg struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type SecurityCfg struct {
	APIKeyPepper          string `mapstructure:"api_key_pepper"`
	APIKeyCacheTTLSeconds int    `mapstructure:"api_key_cache_ttl_seconds"`
}

type IngestCfg struct {
	MaxPayloadBytes int `mapstructure:"max_payload_bytes"`
}

type AgentCfg struct {
	DefaultAPIKeyID string `mapstructure:"default_api_key_id"`
}

type ClerkCfg struct {
	SecretKey   string `mapstructure:"secret_key" yaml:"secret_key"`     // Clerk Secret Key for Backend API
	FrontendURL string `mapstructure:"frontend_url" yaml:"frontend_url"` // Frontend URL for invitation redirect
}

type GrafanaCfg struct {
	URL      string `mapstructure:"url" yaml:"url"`           // Grafana base URL
	Username string `mapstructure:"username" yaml:"username"` // Admin username
	Password string `mapstructure:"password" yaml:"password"` // Admin password
	APIToken string `mapstructure:"api_token" yaml:"api_token"` // API token (alternative to username/password)
}

type Config struct {
	Environment string      `mapstructure:"environment"`
	Server      ServerCfg   `mapstructure:"server"`
	Postgres    PostgresCfg `mapstructure:"postgres"`
	Timescale   PostgresCfg `mapstructure:"timescale"`
	Redis       RedisCfg    `mapstructure:"redis"`
	Security    SecurityCfg `mapstructure:"security"`
	Ingest      IngestCfg   `mapstructure:"ingest"`
	Agent       AgentCfg    `mapstructure:"agent"`
	Clerk       ClerkCfg    `mapstructure:"clerk" yaml:"clerk"`
	Grafana     GrafanaCfg  `mapstructure:"grafana" yaml:"grafana"`
}

func LoadConfig(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func LoadConfigFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	// Override with environment variables
	applyEnvOverrides(&cfg)

	return &cfg, nil
}

// applyEnvOverrides applies environment variable overrides to the config
func applyEnvOverrides(cfg *Config) {
	// Clerk secret key - sensitive, should come from env
	if clerkSecretKey := os.Getenv("CLERK_SECRET_KEY"); clerkSecretKey != "" {
		cfg.Clerk.SecretKey = clerkSecretKey
	}

	// Clerk frontend URL override
	if clerkFrontendURL := os.Getenv("CLERK_FRONTEND_URL"); clerkFrontendURL != "" {
		cfg.Clerk.FrontendURL = clerkFrontendURL
	}

	// API key pepper - sensitive, should come from env
	if apiKeyPepper := os.Getenv("API_KEY_PEPPER"); apiKeyPepper != "" {
		cfg.Security.APIKeyPepper = apiKeyPepper
	}

	// Grafana config - sensitive, should come from env
	if grafanaURL := os.Getenv("GRAFANA_URL"); grafanaURL != "" {
		cfg.Grafana.URL = grafanaURL
	}
	if grafanaUsername := os.Getenv("GRAFANA_USERNAME"); grafanaUsername != "" {
		cfg.Grafana.Username = grafanaUsername
	}
	if grafanaPassword := os.Getenv("GRAFANA_PASSWORD"); grafanaPassword != "" {
		cfg.Grafana.Password = grafanaPassword
	}
	if grafanaAPIToken := os.Getenv("GRAFANA_API_TOKEN"); grafanaAPIToken != "" {
		cfg.Grafana.APIToken = grafanaAPIToken
	}
}
