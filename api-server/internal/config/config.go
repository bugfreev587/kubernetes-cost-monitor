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

type Config struct {
	Environment string      `mapstructure:"environment"`
	Server      ServerCfg   `mapstructure:"server"`
	Postgres    PostgresCfg `mapstructure:"postgres"`
	Timescale   PostgresCfg `mapstructure:"timescale"`
	Redis       RedisCfg    `mapstructure:"redis"`
	Security    SecurityCfg `mapstructure:"security"`
	Ingest      IngestCfg   `mapstructure:"ingest"`
	Agent       AgentCfg    `mapstructure:"agent"`
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
	return &cfg, nil
}
