package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/infrasense/redfish-collector/internal/collector"
	"github.com/infrasense/redfish-collector/internal/config"
	"github.com/infrasense/redfish-collector/internal/metrics"
	_ "github.com/lib/pq"
)

var logger *slog.Logger

func main() {
	// Initialize JSON logger
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting redfish collector", "event", "collector_start", "collector_type", "redfish")

	// Load configuration
	cfg, err := config.LoadConfig("config.yml")
	if err != nil {
		logger.Error("failed to load configuration", "event", "config_load_error", "error", err.Error())
		os.Exit(1)
	}

	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Database,
		cfg.Database.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Error("failed to connect to database", "event", "db_connection_error", "error", err.Error())
		os.Exit(1)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		logger.Error("failed to ping database", "event", "db_ping_error", "error", err.Error())
		os.Exit(1)
	}

	logger.Info("connected to database successfully", "event", "db_connected")

	// Create VictoriaMetrics writer
	metricsWriter := metrics.NewVictoriaMetricsWriter(
		cfg.Metrics.VictoriaMetricsURL,
		cfg.Metrics.BatchSize,
		cfg.GetBatchTimeout(),
	)
	metricsWriter.Start()
	defer metricsWriter.Stop()

	// Encryption key for credential decryption — must match the backend ENCRYPTION_KEY
	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey == "" {
		encryptionKey = cfg.EncryptionKey
	}
	if len(encryptionKey) != 32 {
		logger.Error("ENCRYPTION_KEY must be exactly 32 characters",
			"event", "config_error",
			"length", len(encryptionKey))
		os.Exit(1)
	}

	// Create Redfish collector
	redfishCollector := collector.NewRedfishCollector(
		db,
		metricsWriter,
		encryptionKey,
		cfg.GetPollingInterval(),
		cfg.GetDeviceReloadInterval(),
		cfg.Collector.MaxConcurrent,
		cfg.GetTimeout(),
	)

	// Start collector
	if err := redfishCollector.Start(); err != nil {
		logger.Error("failed to start redfish collector", "event", "collector_start_error", "error", err.Error())
		os.Exit(1)
	}

	// Start health check server
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		deviceCount := redfishCollector.GetDeviceCount()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","device_count":%d}`, deviceCount)
	})

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "# HELP redfish_collector_devices Number of devices being monitored\n")
		fmt.Fprintf(w, "# TYPE redfish_collector_devices gauge\n")
		fmt.Fprintf(w, "redfish_collector_devices %d\n", redfishCollector.GetDeviceCount())
	})

	addr := fmt.Sprintf(":%d", cfg.HealthServer.Port)
	healthServer := &http.Server{
		Addr:         addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		logger.Info("starting health server", "event", "health_server_start", "address", addr)
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("health server failed", "event", "health_server_error", "error", err.Error())
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down redfish collector", "event", "collector_shutdown")
	redfishCollector.Stop()

	// Graceful shutdown of health server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	healthServer.Shutdown(shutdownCtx)

	logger.Info("redfish collector shutdown complete", "event", "collector_shutdown_complete")
}
