package config

import (
	"os"
	"strings"
	"testing"
)

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Database: "infrasense",
			User:     "infrasense",
			Password: "password",
			SSLMode:  "disable",
		},
		Metrics: MetricsConfig{
			VictoriaMetricsURL: "http://localhost:8428",
			BatchSize:          1000,
			BatchTimeout:       "10s",
		},
		Collector: CollectorConfig{
			PollingInterval:      "60s",
			DeviceReloadInterval: "5m",
			MaxConcurrent:        100,
			Timeout:              "30s",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		HealthServer: HealthServerConfig{
			Port: 8080,
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Expected valid config to pass validation, got error: %v", err)
	}
}

func TestValidate_PollingIntervalTooLow(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Database: "infrasense",
			User:     "infrasense",
			Password: "password",
			SSLMode:  "disable",
		},
		Metrics: MetricsConfig{
			VictoriaMetricsURL: "http://localhost:8428",
			BatchSize:          1000,
			BatchTimeout:       "10s",
		},
		Collector: CollectorConfig{
			PollingInterval:      "2s", // Too low (< 5s)
			DeviceReloadInterval: "5m",
			MaxConcurrent:        100,
			Timeout:              "30s",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		HealthServer: HealthServerConfig{
			Port: 8080,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation to fail for polling interval < 5s")
	}
	if !strings.Contains(err.Error(), "collector.polling_interval must be between 5 seconds and 3600 seconds") {
		t.Errorf("Expected error message about polling interval, got: %v", err)
	}
}

func TestValidate_PollingIntervalTooHigh(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Database: "infrasense",
			User:     "infrasense",
			Password: "password",
			SSLMode:  "disable",
		},
		Metrics: MetricsConfig{
			VictoriaMetricsURL: "http://localhost:8428",
			BatchSize:          1000,
			BatchTimeout:       "10s",
		},
		Collector: CollectorConfig{
			PollingInterval:      "7200s", // Too high (> 3600s)
			DeviceReloadInterval: "5m",
			MaxConcurrent:        100,
			Timeout:              "30s",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		HealthServer: HealthServerConfig{
			Port: 8080,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation to fail for polling interval > 3600s")
	}
	if !strings.Contains(err.Error(), "collector.polling_interval must be between 5 seconds and 3600 seconds") {
		t.Errorf("Expected error message about polling interval, got: %v", err)
	}
}

func TestValidate_InvalidPollingInterval(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Database: "infrasense",
			User:     "infrasense",
			Password: "password",
			SSLMode:  "disable",
		},
		Metrics: MetricsConfig{
			VictoriaMetricsURL: "http://localhost:8428",
			BatchSize:          1000,
			BatchTimeout:       "10s",
		},
		Collector: CollectorConfig{
			PollingInterval:      "invalid", // Invalid duration format
			DeviceReloadInterval: "5m",
			MaxConcurrent:        100,
			Timeout:              "30s",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		HealthServer: HealthServerConfig{
			Port: 8080,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation to fail for invalid polling interval format")
	}
	if !strings.Contains(err.Error(), "collector.polling_interval is invalid") {
		t.Errorf("Expected error message about invalid polling interval, got: %v", err)
	}
}

func TestValidate_MissingRequiredFields(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Host: "", // Missing
			Port: 5432,
		},
		Metrics: MetricsConfig{
			VictoriaMetricsURL: "", // Missing
			BatchSize:          0,  // Invalid
		},
		Collector: CollectorConfig{
			PollingInterval:      "60s",
			DeviceReloadInterval: "5m",
			MaxConcurrent:        0, // Invalid
			Timeout:              "30s",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		HealthServer: HealthServerConfig{
			Port: 8080,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation to fail for missing required fields")
	}

	errMsg := err.Error()
	expectedErrors := []string{
		"database.host is required",
		"database.database is required",
		"database.user is required",
		"database.password is required",
		"metrics.victoriametrics_url is required",
		"metrics.batch_size must be greater than 0",
		"collector.max_concurrent must be greater than 0",
	}

	for _, expected := range expectedErrors {
		if !strings.Contains(errMsg, expected) {
			t.Errorf("Expected error message to contain '%s', got: %v", expected, errMsg)
		}
	}
}

func TestLoadConfig_ValidationFailure(t *testing.T) {
	// Create a temporary config file with invalid configuration
	tmpFile, err := os.CreateTemp("", "config-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	invalidConfig := `
database:
  host: ""
  port: 5432
  database: ""
  user: ""
  password: ""
metrics:
  victoriametrics_url: ""
  batch_size: 0
collector:
  polling_interval: "2s"
  device_reload_interval: "5m"
  max_concurrent: 0
  timeout: "30s"
logging:
  level: info
  format: json
health_server:
  port: 8080
`
	if _, err := tmpFile.WriteString(invalidConfig); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	_, err = LoadConfig(tmpFile.Name())
	if err == nil {
		t.Error("Expected LoadConfig to fail for invalid configuration")
	}
	if !strings.Contains(err.Error(), "configuration validation failed") {
		t.Errorf("Expected error message about validation failure, got: %v", err)
	}
}

func TestValidate_EdgeCasePollingIntervals(t *testing.T) {
	tests := []struct {
		name          string
		interval      string
		shouldBeValid bool
	}{
		{"Exactly 5 seconds", "5s", true},
		{"Exactly 3600 seconds", "3600s", true},
		{"Just below 5 seconds", "4999ms", false},
		{"Just above 3600 seconds", "3601s", false},
		{"Valid middle value", "300s", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "infrasense",
					User:     "infrasense",
					Password: "password",
					SSLMode:  "disable",
				},
				Metrics: MetricsConfig{
					VictoriaMetricsURL: "http://localhost:8428",
					BatchSize:          1000,
					BatchTimeout:       "10s",
				},
				Collector: CollectorConfig{
					PollingInterval:      tt.interval,
					DeviceReloadInterval: "5m",
					MaxConcurrent:        100,
					Timeout:              "30s",
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
				HealthServer: HealthServerConfig{
					Port: 8080,
				},
			}

			err := cfg.Validate()
			if tt.shouldBeValid && err != nil {
				t.Errorf("Expected interval '%s' to be valid, got error: %v", tt.interval, err)
			}
			if !tt.shouldBeValid && err == nil {
				t.Errorf("Expected interval '%s' to be invalid, but validation passed", tt.interval)
			}
		})
	}
}
