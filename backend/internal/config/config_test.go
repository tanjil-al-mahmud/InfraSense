package config

import (
	"os"
	"strings"
	"testing"
)

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
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
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Auth: AuthConfig{
			JWTSecret:     "this-is-a-very-long-secret-key-for-jwt-tokens",
			EncryptionKey: "12345678901234567890123456789012", // 32 bytes
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Expected valid config to pass validation, got error: %v", err)
	}
}

func TestValidate_InvalidDatabasePort(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     99999, // Invalid port
			Database: "infrasense",
			User:     "infrasense",
			Password: "password",
			SSLMode:  "disable",
		},
		Metrics: MetricsConfig{
			VictoriaMetricsURL: "http://localhost:8428",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Auth: AuthConfig{
			JWTSecret:     "this-is-a-very-long-secret-key-for-jwt-tokens",
			EncryptionKey: "12345678901234567890123456789012",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation to fail for invalid database port")
	}
	if !strings.Contains(err.Error(), "database.port must be between 1 and 65535") {
		t.Errorf("Expected error message about database port, got: %v", err)
	}
}

func TestValidate_MissingRequiredFields(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Host: "", // Missing required field
			Port: 5432,
		},
		Metrics: MetricsConfig{
			VictoriaMetricsURL: "", // Missing required field
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Auth: AuthConfig{
			JWTSecret:     "short", // Too short
			EncryptionKey: "wrong-length",
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
		"auth.jwt_secret must be at least 32 characters",
		"auth.encryption_key must be exactly 32 bytes",
	}

	for _, expected := range expectedErrors {
		if !strings.Contains(errMsg, expected) {
			t.Errorf("Expected error message to contain '%s', got: %v", expected, errMsg)
		}
	}
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
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
		},
		Logging: LoggingConfig{
			Level:  "invalid", // Invalid log level
			Format: "json",
		},
		Auth: AuthConfig{
			JWTSecret:     "this-is-a-very-long-secret-key-for-jwt-tokens",
			EncryptionKey: "12345678901234567890123456789012",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation to fail for invalid log level")
	}
	if !strings.Contains(err.Error(), "logging.level must be one of: debug, info, warn, error") {
		t.Errorf("Expected error message about log level, got: %v", err)
	}
}

func TestLoad_ValidationFailure(t *testing.T) {
	// Create a temporary config file with invalid configuration
	tmpFile, err := os.CreateTemp("", "config-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	invalidConfig := `
server:
  host: localhost
  port: 99999
database:
  host: ""
  port: 5432
  database: ""
  user: ""
  password: ""
metrics:
  victoriametrics_url: ""
logging:
  level: info
  format: json
auth:
  jwt_secret: "short"
  encryption_key: "wrong"
`
	if _, err := tmpFile.WriteString(invalidConfig); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	_, err = Load(tmpFile.Name())
	if err == nil {
		t.Error("Expected Load to fail for invalid configuration")
	}
	if !strings.Contains(err.Error(), "configuration validation failed") {
		t.Errorf("Expected error message about validation failure, got: %v", err)
	}
}
