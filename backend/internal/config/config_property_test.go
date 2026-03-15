package config

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty_ConfigurationValidation validates that invalid configurations
// are always rejected with non-zero exit behavior
// Property 35: Configuration Validation
// Validates: Requirements 28.3
func TestProperty_ConfigurationValidation(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Invalid database port should always be rejected
	properties.Property("Invalid database port rejected", prop.ForAll(
		func(port int) bool {
			cfg := &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 8080,
				},
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     port,
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
			// Invalid ports (< 1 or > 65535) should always produce an error
			if port < 1 || port > 65535 {
				return err != nil
			}
			// Valid ports should not produce an error
			return err == nil
		},
		gen.IntRange(-1000, 70000),
	))

	// Property: Missing required database fields should always be rejected
	properties.Property("Missing database host rejected", prop.ForAll(
		func(host string) bool {
			cfg := &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 8080,
				},
				Database: DatabaseConfig{
					Host:     host,
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
					EncryptionKey: "12345678901234567890123456789012",
				},
			}

			err := cfg.Validate()
			// Empty host should always produce an error
			if host == "" {
				return err != nil
			}
			// Non-empty host should not produce an error
			return err == nil
		},
		gen.OneConstOf("", "localhost", "db.example.com", "192.168.1.1"),
	))

	// Property: Invalid JWT secret length should always be rejected
	properties.Property("Short JWT secret rejected", prop.ForAll(
		func(secretLen int) bool {
			secret := ""
			for i := 0; i < secretLen; i++ {
				secret += "a"
			}

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
					JWTSecret:     secret,
					EncryptionKey: "12345678901234567890123456789012",
				},
			}

			err := cfg.Validate()
			// JWT secret < 32 characters should always produce an error
			if secretLen < 32 {
				return err != nil
			}
			// JWT secret >= 32 characters should not produce an error
			return err == nil
		},
		gen.IntRange(0, 64),
	))

	// Property: Invalid encryption key length should always be rejected
	properties.Property("Wrong encryption key length rejected", prop.ForAll(
		func(keyLen int) bool {
			key := ""
			for i := 0; i < keyLen; i++ {
				key += "a"
			}

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
					EncryptionKey: key,
				},
			}

			err := cfg.Validate()
			// Encryption key != 32 bytes should always produce an error
			if keyLen != 32 {
				return err != nil
			}
			// Encryption key == 32 bytes should not produce an error
			return err == nil
		},
		gen.IntRange(0, 64),
	))

	// Property: Invalid log level should always be rejected
	properties.Property("Invalid log level rejected", prop.ForAll(
		func(logLevel string) bool {
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
					Level:  logLevel,
					Format: "json",
				},
				Auth: AuthConfig{
					JWTSecret:     "this-is-a-very-long-secret-key-for-jwt-tokens",
					EncryptionKey: "12345678901234567890123456789012",
				},
			}

			err := cfg.Validate()
			validLevels := map[string]bool{
				"debug": true,
				"info":  true,
				"warn":  true,
				"error": true,
			}

			// Invalid log levels should always produce an error
			if !validLevels[logLevel] {
				return err != nil
			}
			// Valid log levels should not produce an error
			return err == nil
		},
		gen.OneConstOf("debug", "info", "warn", "error", "invalid", "trace", "fatal", ""),
	))

	// Property: Missing VictoriaMetrics URL should always be rejected
	properties.Property("Missing VictoriaMetrics URL rejected", prop.ForAll(
		func(url string) bool {
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
					VictoriaMetricsURL: url,
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
			// Empty URL should always produce an error
			if url == "" {
				return err != nil
			}
			// Non-empty URL should not produce an error
			return err == nil
		},
		gen.OneConstOf("", "http://localhost:8428", "http://victoriametrics:8428"),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty_ConfigurationValidation_ExitBehavior validates that services
// exit with non-zero status code when configuration is invalid
func TestProperty_ConfigurationValidation_ExitBehavior(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Invalid configuration should cause validation error
	properties.Property("Invalid config produces validation error", prop.ForAll(
		func(dbPort int, jwtSecretLen int, encKeyLen int) bool {
			// Build JWT secret and encryption key of specified lengths
			jwtSecret := ""
			for i := 0; i < jwtSecretLen; i++ {
				jwtSecret += "a"
			}
			encKey := ""
			for i := 0; i < encKeyLen; i++ {
				encKey += "b"
			}

			cfg := &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 8080,
				},
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     dbPort,
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
					JWTSecret:     jwtSecret,
					EncryptionKey: encKey,
				},
			}

			err := cfg.Validate()

			// Determine if configuration is valid
			isValidPort := dbPort >= 1 && dbPort <= 65535
			isValidJWT := jwtSecretLen >= 32
			isValidEncKey := encKeyLen == 32

			isValid := isValidPort && isValidJWT && isValidEncKey

			// If configuration is invalid, error should be non-nil
			if !isValid {
				return err != nil
			}
			// If configuration is valid, error should be nil
			return err == nil
		},
		gen.IntRange(0, 70000), // dbPort
		gen.IntRange(0, 64),    // jwtSecretLen
		gen.IntRange(0, 64),    // encKeyLen
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty_PollingIntervalValidation validates that polling intervals
// are constrained to valid ranges
func TestProperty_PollingIntervalValidation(t *testing.T) {
	// Note: Backend config doesn't have polling interval validation
	// This property is validated in collector configs
	t.Skip("Polling interval validation is specific to collector configs")
}
