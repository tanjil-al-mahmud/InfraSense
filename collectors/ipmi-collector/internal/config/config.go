package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database     DatabaseConfig     `yaml:"database"`
	Metrics      MetricsConfig      `yaml:"metrics"`
	Collector    CollectorConfig    `yaml:"collector"`
	Logging      LoggingConfig      `yaml:"logging"`
	HealthServer HealthServerConfig `yaml:"health_server"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"ssl_mode"`
}

type MetricsConfig struct {
	VictoriaMetricsURL string `yaml:"victoriametrics_url"`
	BatchSize          int    `yaml:"batch_size"`
	BatchTimeout       string `yaml:"batch_timeout"`
}

type CollectorConfig struct {
	PollingInterval      string `yaml:"polling_interval"`
	DeviceReloadInterval string `yaml:"device_reload_interval"`
	MaxConcurrent        int    `yaml:"max_concurrent"`
	Timeout              string `yaml:"timeout"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type HealthServerConfig struct {
	Port int `yaml:"port"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		cfg.Database.Host = dbHost
	}
	if dbPassword := os.Getenv("DB_PASSWORD"); dbPassword != "" {
		cfg.Database.Password = dbPassword
	}
	if vmURL := os.Getenv("VICTORIAMETRICS_URL"); vmURL != "" {
		cfg.Metrics.VictoriaMetricsURL = vmURL
	}

	// Set defaults
	if cfg.Collector.MaxConcurrent == 0 {
		cfg.Collector.MaxConcurrent = 100
	}
	if cfg.Collector.PollingInterval == "" {
		cfg.Collector.PollingInterval = "60s"
	}
	if cfg.Collector.DeviceReloadInterval == "" {
		cfg.Collector.DeviceReloadInterval = "5m"
	}
	if cfg.Collector.Timeout == "" {
		cfg.Collector.Timeout = "30s"
	}
	if cfg.Metrics.BatchSize == 0 {
		cfg.Metrics.BatchSize = 1000
	}
	if cfg.Metrics.BatchTimeout == "" {
		cfg.Metrics.BatchTimeout = "10s"
	}
	if cfg.HealthServer.Port == 0 {
		cfg.HealthServer.Port = 8080
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

func (c *Config) GetPollingInterval() time.Duration {
	d, _ := time.ParseDuration(c.Collector.PollingInterval)
	return d
}

func (c *Config) GetDeviceReloadInterval() time.Duration {
	d, _ := time.ParseDuration(c.Collector.DeviceReloadInterval)
	return d
}

func (c *Config) GetTimeout() time.Duration {
	d, _ := time.ParseDuration(c.Collector.Timeout)
	return d
}

func (c *Config) GetBatchTimeout() time.Duration {
	d, _ := time.ParseDuration(c.Metrics.BatchTimeout)
	return d
}

// Validate validates the configuration
func (c *Config) Validate() error {
	var errors []string

	// Validate database configuration
	if c.Database.Host == "" {
		errors = append(errors, "database.host is required")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		errors = append(errors, "database.port must be between 1 and 65535")
	}
	if c.Database.Database == "" {
		errors = append(errors, "database.database is required")
	}
	if c.Database.User == "" {
		errors = append(errors, "database.user is required")
	}
	if c.Database.Password == "" {
		errors = append(errors, "database.password is required")
	}

	// Validate metrics configuration
	if c.Metrics.VictoriaMetricsURL == "" {
		errors = append(errors, "metrics.victoriametrics_url is required")
	}
	if c.Metrics.BatchSize <= 0 {
		errors = append(errors, "metrics.batch_size must be greater than 0")
	}

	// Validate polling interval (must be between 5s and 3600s)
	pollingInterval, err := time.ParseDuration(c.Collector.PollingInterval)
	if err != nil {
		errors = append(errors, fmt.Sprintf("collector.polling_interval is invalid: %v", err))
	} else if pollingInterval < 5*time.Second || pollingInterval > 3600*time.Second {
		errors = append(errors, "collector.polling_interval must be between 5 seconds and 3600 seconds")
	}

	// Validate device reload interval
	if _, err := time.ParseDuration(c.Collector.DeviceReloadInterval); err != nil {
		errors = append(errors, fmt.Sprintf("collector.device_reload_interval is invalid: %v", err))
	}

	// Validate timeout
	if _, err := time.ParseDuration(c.Collector.Timeout); err != nil {
		errors = append(errors, fmt.Sprintf("collector.timeout is invalid: %v", err))
	}

	// Validate batch timeout
	if _, err := time.ParseDuration(c.Metrics.BatchTimeout); err != nil {
		errors = append(errors, fmt.Sprintf("metrics.batch_timeout is invalid: %v", err))
	}

	// Validate max concurrent
	if c.Collector.MaxConcurrent <= 0 {
		errors = append(errors, "collector.max_concurrent must be greater than 0")
	}

	// Validate health server port
	if c.HealthServer.Port <= 0 || c.HealthServer.Port > 65535 {
		errors = append(errors, "health_server.port must be between 1 and 65535")
	}

	// Validate logging configuration
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if c.Logging.Level != "" && !validLogLevels[c.Logging.Level] {
		errors = append(errors, "logging.level must be one of: debug, info, warn, error")
	}

	validLogFormats := map[string]bool{"json": true, "text": true}
	if c.Logging.Format != "" && !validLogFormats[c.Logging.Format] {
		errors = append(errors, "logging.format must be one of: json, text")
	}

	if len(errors) > 0 {
		return fmt.Errorf("\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}
