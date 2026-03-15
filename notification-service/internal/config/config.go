package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Email    EmailConfig    `yaml:"email"`
	Telegram TelegramConfig `yaml:"telegram"`
	Slack    SlackConfig    `yaml:"slack"`
	Logging  LoggingConfig  `yaml:"logging"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type EmailConfig struct {
	Enabled  bool   `yaml:"enabled"`
	SMTPHost string `yaml:"smtp_host"`
	SMTPPort int    `yaml:"smtp_port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
	To       string `yaml:"to"`
	UseTLS   bool   `yaml:"use_tls"`
}

type TelegramConfig struct {
	Enabled  bool   `yaml:"enabled"`
	BotToken string `yaml:"bot_token"`
	ChatID   string `yaml:"chat_id"`
}

type SlackConfig struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
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
	if smtpHost := os.Getenv("SMTP_HOST"); smtpHost != "" {
		cfg.Email.SMTPHost = smtpHost
	}
	if smtpPassword := os.Getenv("SMTP_PASSWORD"); smtpPassword != "" {
		cfg.Email.Password = smtpPassword
	}
	if telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN"); telegramToken != "" {
		cfg.Telegram.BotToken = telegramToken
	}
	if telegramChatID := os.Getenv("TELEGRAM_CHAT_ID"); telegramChatID != "" {
		cfg.Telegram.ChatID = telegramChatID
	}
	if slackWebhook := os.Getenv("SLACK_WEBHOOK_URL"); slackWebhook != "" {
		cfg.Slack.WebhookURL = slackWebhook
	}

	// Set defaults
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	var errors []string

	// Validate server configuration
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		errors = append(errors, "server.port must be between 1 and 65535")
	}

	// Validate email configuration if enabled
	if c.Email.Enabled {
		if c.Email.SMTPHost == "" {
			errors = append(errors, "email.smtp_host is required when email is enabled")
		}
		if c.Email.SMTPPort <= 0 || c.Email.SMTPPort > 65535 {
			errors = append(errors, "email.smtp_port must be between 1 and 65535 when email is enabled")
		}
		if c.Email.From == "" {
			errors = append(errors, "email.from is required when email is enabled")
		}
		if c.Email.To == "" {
			errors = append(errors, "email.to is required when email is enabled")
		}
	}

	// Validate Telegram configuration if enabled
	if c.Telegram.Enabled {
		if c.Telegram.BotToken == "" {
			errors = append(errors, "telegram.bot_token is required when telegram is enabled")
		}
		if c.Telegram.ChatID == "" {
			errors = append(errors, "telegram.chat_id is required when telegram is enabled")
		}
	}

	// Validate Slack configuration if enabled
	if c.Slack.Enabled {
		if c.Slack.WebhookURL == "" {
			errors = append(errors, "slack.webhook_url is required when slack is enabled")
		}
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
