package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/infrasense/notification-service/internal/config"
	"github.com/infrasense/notification-service/internal/notifier"
	"github.com/infrasense/notification-service/internal/webhook"
)

var logger *slog.Logger

func main() {
	// Initialize JSON logger
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting notification service", "event", "service_start")

	// Load configuration
	cfg, err := config.LoadConfig("config.yml")
	if err != nil {
		logger.Error("failed to load configuration", "event", "config_load_error", "error", err.Error())
		os.Exit(1)
	}

	// Initialize notifiers
	notifiers := make([]webhook.Notifier, 0)

	if cfg.Email.Enabled {
		emailNotifier := notifier.NewEmailNotifier(
			cfg.Email.SMTPHost,
			cfg.Email.SMTPPort,
			cfg.Email.Username,
			cfg.Email.Password,
			cfg.Email.From,
			cfg.Email.To,
			cfg.Email.UseTLS,
		)
		notifiers = append(notifiers, emailNotifier)
		logger.Info("email notifier enabled", "event", "notifier_enabled", "channel", "email")
	}

	if cfg.Telegram.Enabled {
		telegramNotifier := notifier.NewTelegramNotifier(
			cfg.Telegram.BotToken,
			cfg.Telegram.ChatID,
		)
		notifiers = append(notifiers, telegramNotifier)
		logger.Info("telegram notifier enabled", "event", "notifier_enabled", "channel", "telegram")
	}

	if cfg.Slack.Enabled {
		slackNotifier := notifier.NewSlackNotifier(cfg.Slack.WebhookURL)
		notifiers = append(notifiers, slackNotifier)
		logger.Info("slack notifier enabled", "event", "notifier_enabled", "channel", "slack")
	}

	if len(notifiers) == 0 {
		logger.Warn("no notifiers enabled", "event", "no_notifiers")
	}

	// Create webhook handler
	handler := webhook.NewHandler(notifiers)

	// Setup HTTP server
	http.HandleFunc("/webhook", handler.HandleWebhook)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","notifiers":%d}`, len(notifiers))
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("notification service listening", "event", "server_start", "address", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "event", "server_error", "error", err.Error())
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down notification service", "event", "service_shutdown")

	// Graceful shutdown with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "event", "shutdown_error", "error", err.Error())
		os.Exit(1)
	}

	logger.Info("notification service shutdown complete", "event", "service_shutdown_complete")
}
