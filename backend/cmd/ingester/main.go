package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/infrasense/backend/internal/config"
	"github.com/infrasense/backend/internal/db"
	"github.com/infrasense/backend/internal/ingest"
	"github.com/infrasense/backend/internal/metrics"
	"github.com/nats-io/nats.go"
)

func main() {
	cfg, err := config.Load("config.yml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	database, err := db.NewDB(db.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		Database: cfg.Database.Database,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		SSLMode:  cfg.Database.SSLMode,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	log.Println("Validating database schema and applying migrations...")
	if err := database.RunMigrations("migrations"); err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}

	natsURL := ingest.EnvOrDefault("NATS_URL", "nats://nats:4222")
	nc, err := nats.Connect(natsURL, nats.Name("infrasense-ingester"))
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Failed to init JetStream: %v", err)
	}

	if err := ingest.EnsureStreams(js); err != nil {
		log.Fatalf("Failed to ensure streams: %v", err)
	}

	vmURL := os.Getenv("VICTORIAMETRICS_URL")
	if vmURL == "" {
		vmURL = "http://victoriametrics:8428/api/v1/write"
	}
	batchSize := envInt("VM_BATCH_SIZE", 1000)
	batchTimeout := envDuration("VM_BATCH_TIMEOUT", 10*time.Second)

	vmWriter := metrics.NewVictoriaMetricsWriter(vmURL, batchSize, batchTimeout)
	eventRepo := db.NewHardwareEventRepository(database)
	ing := ingest.New(js, eventRepo, vmWriter)

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := ing.Run(runCtx, ingest.Config{
			NATSURL:            natsURL,
			VictoriaWriteURL:   vmURL,
			VMBatchSize:        batchSize,
			VMBatchTimeout:     batchTimeout,
			DurableNameMetrics: "INFRA_INGEST_METRICS",
			DurableNameEvents:  "INFRA_INGEST_EVENTS",
		}); err != nil {
			log.Printf("Ingester stopped with error: %v", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

